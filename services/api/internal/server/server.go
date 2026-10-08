package server

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	pb "github.com/architsangal/AteThat/services/api/internal/gen/atethat/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Config struct {
	CoreAddress    string
	FrontendOrigin string
	VerifyKey      ed25519.PublicKey
	InternalSecret []byte
	CookieSecure   bool
}

type Server struct {
	Config   Config
	Conn     *grpc.ClientConn
	Identity pb.IdentityServiceClient
	Profile  pb.ProfileServiceClient
}

func New(ctx context.Context, cfg Config) (*Server, error) {
	if cfg.CoreAddress == "" || cfg.FrontendOrigin == "" || len(cfg.VerifyKey) != ed25519.PublicKeySize || len(cfg.InternalSecret) < 32 {
		return nil, errors.New("invalid API configuration")
	}
	conn, err := grpc.DialContext(ctx, cfg.CoreAddress, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, err
	}
	return &Server{Config: cfg, Conn: conn, Identity: pb.NewIdentityServiceClient(conn), Profile: pb.NewProfileServiceClient(conn)}, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("X-Request-Id")
	if len(requestID) < 8 || len(requestID) > 64 || strings.ContainsAny(requestID, " \r\n") {
		requestID = randomID()
	}
	w.Header().Set("X-Request-Id", requestID)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	if origin := r.Header.Get("Origin"); origin != "" && origin == s.Config.FrontendOrigin {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-Id")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
	}
	if r.Method == http.MethodOptions {
		if r.Header.Get("Origin") != s.Config.FrontendOrigin {
			s.problem(w, requestID, http.StatusForbidden, "ORIGIN_NOT_ALLOWED", "Origin not allowed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.URL.Path == "/healthz" && r.Method == http.MethodGet {
		s.json(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
		s.problem(w, requestID, http.StatusNotFound, "RESOURCE_NOT_FOUND", "Route not found")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1")
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	switch {
	case path == "/auth/register" && r.Method == http.MethodPost:
		s.register(w, r, requestID, ctx)
	case path == "/auth/verify-email" && r.Method == http.MethodPost:
		s.verifyEmail(w, r, requestID, ctx)
	case path == "/auth/resend-verification" && r.Method == http.MethodPost:
		s.emailAction(w, r, requestID, ctx, "resend")
	case path == "/auth/login" && r.Method == http.MethodPost:
		s.login(w, r, requestID, ctx)
	case path == "/auth/google/start" && r.Method == http.MethodGet:
		s.startGoogle(w, r, requestID, ctx)
	case path == "/auth/google/callback" && r.Method == http.MethodGet:
		s.completeGoogle(w, r, requestID, ctx)
	case path == "/account/identities/google/link/start" && r.Method == http.MethodGet:
		s.startGoogleLink(w, r, requestID, ctx)
	case path == "/auth/refresh" && r.Method == http.MethodPost:
		s.refresh(w, r, requestID, ctx)
	case path == "/auth/logout" && r.Method == http.MethodPost:
		s.logout(w, r, requestID, ctx)
	case path == "/auth/forgot-password" && r.Method == http.MethodPost:
		s.emailAction(w, r, requestID, ctx, "reset")
	case path == "/auth/reset-password" && r.Method == http.MethodPost:
		s.resetPassword(w, r, requestID, ctx)
	case path == "/profile" && r.Method == http.MethodGet:
		s.getProfile(w, r, requestID, ctx)
	case path == "/profile" && r.Method == http.MethodPatch:
		s.updateProfile(w, r, requestID, ctx)
	case path == "/preferences" && r.Method == http.MethodGet:
		s.getPreferences(w, r, requestID, ctx)
	case path == "/preferences" && r.Method == http.MethodPatch:
		s.updatePreferences(w, r, requestID, ctx)
	default:
		s.problem(w, requestID, http.StatusNotFound, "RESOURCE_NOT_FOUND", "Route not found")
	}
}

func randomID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(value)
}

func (s *Server) rpcContext(ctx context.Context, method, userID string, family ...string) context.Context {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := randomID()
	sessionFamily := ""
	if len(family) == 1 {
		sessionFamily = family[0]
	}
	mac := hmac.New(sha256.New, s.Config.InternalSecret)
	_, _ = fmt.Fprintf(mac, "%s|%s|%s|%s|%s", method, ts, userID, sessionFamily, nonce)
	return metadata.AppendToOutgoingContext(ctx, "x-atethat-ts", ts, "x-atethat-user", userID,
		"x-atethat-session", sessionFamily, "x-atethat-nonce", nonce, "x-atethat-sig", hex.EncodeToString(mac.Sum(nil)))
}

type principal struct {
	UserID        string
	SessionFamily string
}

func (s *Server) principal(r *http.Request) (principal, error) {
	parts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return principal{}, errors.New("missing bearer token")
	}
	return verifyAccessToken(parts[1], s.Config.VerifyKey)
}

func verifyAccessToken(token string, key ed25519.PublicKey) (principal, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return principal{}, errors.New("invalid token")
	}
	headerRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return principal{}, err
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerRaw, &header); err != nil || header.Alg != "EdDSA" || header.Kid != "local-v1" {
		return principal{}, errors.New("invalid token header")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !ed25519.Verify(key, []byte(parts[0]+"."+parts[1]), signature) {
		return principal{}, errors.New("invalid token signature")
	}
	claimsRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return principal{}, err
	}
	var claims struct {
		Iss string `json:"iss"`
		Aud string `json:"aud"`
		Sub string `json:"sub"`
		Sid string `json:"sid"`
		Exp int64  `json:"exp"`
		Iat int64  `json:"iat"`
	}
	if err := json.Unmarshal(claimsRaw, &claims); err != nil {
		return principal{}, err
	}
	now := time.Now().Unix()
	if claims.Iss != "atethat-core" || claims.Aud != "atethat-api" || len(claims.Sub) != 24 || claims.Sid == "" || claims.Exp <= now || claims.Iat > now+30 {
		return principal{}, errors.New("invalid token claims")
	}
	return principal{UserID: claims.Sub, SessionFamily: claims.Sid}, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target interface{}) error {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return errors.New("application/json is required")
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(interface{})); err != io.EOF {
		return errors.New("only one JSON object is allowed")
	}
	return nil
}

func (s *Server) json(w http.ResponseWriter, code int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}

func (s *Server) problem(w http.ResponseWriter, requestID string, httpCode int, code, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(httpCode)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"type":  "https://errors.atethat.invalid/" + strings.ToLower(strings.ReplaceAll(code, "_", "-")),
		"title": http.StatusText(httpCode), "status": httpCode, "code": code, "detail": detail, "request_id": requestID,
	})
}

func (s *Server) rpcProblem(w http.ResponseWriter, requestID string, err error) {
	code := status.Code(err)
	switch code {
	case codes.InvalidArgument:
		s.problem(w, requestID, 422, "VALIDATION_FAILED", "One or more fields are invalid")
	case codes.Unauthenticated:
		s.problem(w, requestID, 401, "UNAUTHENTICATED", "Authentication failed")
	case codes.PermissionDenied:
		s.problem(w, requestID, 403, "EMAIL_NOT_VERIFIED", "Account unavailable or email not verified")
	case codes.NotFound:
		s.problem(w, requestID, 404, "RESOURCE_NOT_FOUND", "Resource not found")
	case codes.Aborted:
		s.problem(w, requestID, 409, "VERSION_CONFLICT", "Resource has changed")
	case codes.FailedPrecondition:
		s.problem(w, requestID, 410, "TOKEN_EXPIRED", "Token expired or already used")
	default:
		s.problem(w, requestID, 503, "CORE_UNAVAILABLE", "Service temporarily unavailable")
	}
}
