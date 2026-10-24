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
	"log"
	"net/http"
	"net/netip"
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
	CoreAddress       string
	FrontendOrigin    string
	VerifyKey         ed25519.PublicKey
	InternalSecret    []byte
	CookieSecure      bool
	TrustedProxyCIDRs []netip.Prefix
}

type Server struct {
	Config      Config
	Conn        *grpc.ClientConn
	Identity    pb.IdentityServiceClient
	Profile     pb.ProfileServiceClient
	Target      pb.TargetServiceClient
	Food        pb.FoodServiceClient
	FoodLog     pb.FoodLogServiceClient
	Dashboard   pb.DashboardServiceClient
	Health      pb.HealthServiceClient
	Analysis    pb.AnalysisServiceClient
	Wellness    pb.WellnessServiceClient
	authLimiter rateLimiter
}

func New(ctx context.Context, cfg Config) (*Server, error) {
	if cfg.CoreAddress == "" || cfg.FrontendOrigin == "" || len(cfg.VerifyKey) != ed25519.PublicKeySize || len(cfg.InternalSecret) < 32 {
		return nil, errors.New("invalid API configuration")
	}
	conn, err := grpc.DialContext(ctx, cfg.CoreAddress, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, err
	}
	return &Server{Config: cfg, Conn: conn, Identity: pb.NewIdentityServiceClient(conn),
		Profile: pb.NewProfileServiceClient(conn), Target: pb.NewTargetServiceClient(conn), Food: pb.NewFoodServiceClient(conn),
		FoodLog: pb.NewFoodLogServiceClient(conn), Dashboard: pb.NewDashboardServiceClient(conn),
		Health: pb.NewHealthServiceClient(conn), Analysis: pb.NewAnalysisServiceClient(conn), Wellness: pb.NewWellnessServiceClient(conn)}, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()
	recorded := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
	w = recorded
	requestID := r.Header.Get("X-Request-Id")
	if len(requestID) < 8 || len(requestID) > 64 || strings.ContainsAny(requestID, " \r\n") {
		requestID = randomID()
	}
	defer func() {
		if r.URL.Path == "/healthz" {
			return
		}
		entry, _ := json.Marshal(map[string]interface{}{"event": "http_request", "request_id": requestID,
			"method": r.Method, "path": r.URL.Path, "status": recorded.status,
			"duration_ms": time.Since(startedAt).Milliseconds()})
		log.Print(string(entry))
	}()
	w.Header().Set("X-Request-Id", requestID)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	if origin := r.Header.Get("Origin"); origin != "" && origin == s.Config.FrontendOrigin {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-Id, Idempotency-Key")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	}
	if r.Method == http.MethodOptions {
		if r.Header.Get("Origin") != s.Config.FrontendOrigin {
			s.problem(w, requestID, http.StatusForbidden, "ORIGIN_NOT_ALLOWED", "Origin not allowed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.URL.Path == "/healthz" && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		s.json(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
		s.problem(w, requestID, http.StatusNotFound, "RESOURCE_NOT_FOUND", "Route not found")
		return
	}
	if !s.limitPublicAuth(w, r, requestID) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1")
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	switch {
	case path == "/wellness/records" || strings.HasPrefix(path, "/wellness/records/") || path == "/wellness/settings" || path == "/wellness/maintenance" || path == "/wellness/analysis":
		s.wellnessRoute(w, r, requestID, ctx, path)
	case path == "/auth/register" && r.Method == http.MethodPost:
		s.register(w, r, requestID, ctx)
	case path == "/auth/verify-email" && r.Method == http.MethodPost:
		s.verifyEmail(w, r, requestID, ctx)
	case path == "/auth/email-login" && r.Method == http.MethodPost:
		s.completeEmailSignIn(w, r, requestID, ctx)
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
	case path == "/auth/google/cancel-deletion/start" && r.Method == http.MethodGet:
		s.startGoogleDeletionCancellation(w, r, requestID, ctx)
	case path == "/account/deletion-request/google/start" && r.Method == http.MethodGet:
		s.startGoogleDeletionRequest(w, r, requestID, ctx)
	case path == "/auth/refresh" && r.Method == http.MethodPost:
		s.refresh(w, r, requestID, ctx)
	case path == "/auth/logout" && r.Method == http.MethodPost:
		s.logout(w, r, requestID, ctx)
	case path == "/auth/forgot-password" && r.Method == http.MethodPost:
		s.emailAction(w, r, requestID, ctx, "reset")
	case path == "/auth/reset-password" && r.Method == http.MethodPost:
		s.resetPassword(w, r, requestID, ctx)
	case path == "/account/deletion-request" && r.Method == http.MethodPost:
		s.requestAccountDeletion(w, r, requestID, ctx)
	case path == "/auth/cancel-deletion" && r.Method == http.MethodPost:
		s.cancelAccountDeletion(w, r, requestID, ctx)
	case path == "/profile" && r.Method == http.MethodGet:
		s.getProfile(w, r, requestID, ctx)
	case path == "/profile" && r.Method == http.MethodPatch:
		s.updateProfile(w, r, requestID, ctx)
	case path == "/preferences" && r.Method == http.MethodGet:
		s.getPreferences(w, r, requestID, ctx)
	case path == "/preferences" && r.Method == http.MethodPatch:
		s.updatePreferences(w, r, requestID, ctx)
	case path == "/tracking-day/current" && r.Method == http.MethodGet:
		s.getCurrentTrackingDay(w, r, requestID, ctx)
	case path == "/nutrition/targets/planning" && (r.Method == http.MethodGet || r.Method == http.MethodPost):
		s.targetPlanning(w, r, requestID, ctx)
	case path == "/nutrition/targets/custom" && r.Method == http.MethodPost:
		s.saveCustomTargets(w, r, requestID, ctx)
	case path == "/nutrition/targets/custom/review" && r.Method == http.MethodPost:
		s.saveCustomTargets(w, r, requestID, ctx)
	case path == "/nutrition/targets/proposal" && r.Method == http.MethodPost:
		s.proposeCalculatedTargets(w, r, requestID, ctx)
	case path == "/nutrition/targets/reset" && r.Method == http.MethodPost:
		s.resetCalculatedTargets(w, r, requestID, ctx)
	case path == "/text-summary" && r.Method == http.MethodGet:
		s.textSummary(w, r, requestID, ctx)
	case path == "/nutrition/targets/calculate" && r.Method == http.MethodPost:
		s.calculateTargets(w, r, requestID, ctx)
	case path == "/nutrition/targets" && r.Method == http.MethodGet:
		s.getTargets(w, r, requestID, ctx)
	case path == "/nutrition/targets/history" && r.Method == http.MethodGet:
		s.targetHistory(w, r, requestID, ctx)
	case strings.HasPrefix(path, "/nutrition/targets/") && strings.HasSuffix(path, "/accept") && r.Method == http.MethodPost:
		s.acceptTargets(w, r, requestID, ctx, strings.TrimSuffix(strings.TrimPrefix(path, "/nutrition/targets/"), "/accept"))
	case strings.HasPrefix(path, "/nutrition/targets/") && strings.HasSuffix(path, "/reject") && r.Method == http.MethodPost:
		s.rejectTargets(w, r, requestID, ctx, strings.TrimSuffix(strings.TrimPrefix(path, "/nutrition/targets/"), "/reject"))
	case path == "/foods/search" && r.Method == http.MethodGet:
		s.searchFoods(w, r, requestID, ctx)
	case strings.HasPrefix(path, "/foods/provider/") && r.Method == http.MethodGet:
		s.getProviderFood(w, r, requestID, ctx, strings.TrimPrefix(path, "/foods/provider/"))
	case path == "/foods/favourites" && r.Method == http.MethodGet:
		s.listFavourites(w, r, requestID, ctx)
	case path == "/foods/favourites" && r.Method == http.MethodPost:
		s.addFavourite(w, r, requestID, ctx)
	case strings.HasPrefix(path, "/foods/favourites/") && r.Method == http.MethodDelete:
		s.deleteFavourite(w, r, requestID, ctx, strings.TrimPrefix(path, "/foods/favourites/"))
	case path == "/foods/recent" && r.Method == http.MethodGet:
		s.listUsedFoods(w, r, requestID, ctx, "RECENT")
	case path == "/foods/frequent" && r.Method == http.MethodGet:
		s.listUsedFoods(w, r, requestID, ctx, "FREQUENT")
	case path == "/foods/custom" && r.Method == http.MethodGet:
		s.listCustomFoods(w, r, requestID, ctx)
	case path == "/foods/custom" && r.Method == http.MethodPost:
		s.createCustomFood(w, r, requestID, ctx)
	case strings.HasPrefix(path, "/foods/custom/") && r.Method == http.MethodGet:
		s.getCustomFood(w, r, requestID, ctx, strings.TrimPrefix(path, "/foods/custom/"))
	case strings.HasPrefix(path, "/foods/custom/") && r.Method == http.MethodPatch:
		s.updateCustomFood(w, r, requestID, ctx, strings.TrimPrefix(path, "/foods/custom/"))
	case strings.HasPrefix(path, "/foods/custom/") && r.Method == http.MethodDelete:
		s.deleteCustomFood(w, r, requestID, ctx, strings.TrimPrefix(path, "/foods/custom/"))
	case path == "/meals" && r.Method == http.MethodGet:
		s.listMeals(w, r, requestID, ctx)
	case path == "/servings" && r.Method == http.MethodGet:
		s.listServings(w, r, requestID, ctx)
	case path == "/servings" && r.Method == http.MethodPost:
		s.createServing(w, r, requestID, ctx)
	case strings.HasPrefix(path, "/servings/") && r.Method == http.MethodGet:
		s.getServing(w, r, requestID, ctx, strings.TrimPrefix(path, "/servings/"))
	case strings.HasPrefix(path, "/servings/") && r.Method == http.MethodPatch:
		s.updateServing(w, r, requestID, ctx, strings.TrimPrefix(path, "/servings/"))
	case strings.HasPrefix(path, "/servings/") && r.Method == http.MethodDelete:
		s.deleteServing(w, r, requestID, ctx, strings.TrimPrefix(path, "/servings/"))
	case path == "/meals" && r.Method == http.MethodPost:
		s.createMeal(w, r, requestID, ctx)
	case strings.HasPrefix(path, "/meals/") && strings.HasSuffix(path, "/preview") && r.Method == http.MethodPost:
		s.previewMeal(w, r, requestID, ctx, strings.TrimSuffix(strings.TrimPrefix(path, "/meals/"), "/preview"))
	case strings.HasPrefix(path, "/meals/") && r.Method == http.MethodGet:
		s.getMeal(w, r, requestID, ctx, strings.TrimPrefix(path, "/meals/"))
	case strings.HasPrefix(path, "/meals/") && r.Method == http.MethodPatch:
		s.updateMeal(w, r, requestID, ctx, strings.TrimPrefix(path, "/meals/"))
	case strings.HasPrefix(path, "/meals/") && r.Method == http.MethodDelete:
		s.deleteMeal(w, r, requestID, ctx, strings.TrimPrefix(path, "/meals/"))
	case path == "/food-log-groups/preview" && r.Method == http.MethodPost:
		s.previewFoodGroup(w, r, requestID, ctx)
	case path == "/food-log-groups/confirm" && r.Method == http.MethodPost:
		s.confirmFoodGroup(w, r, requestID, ctx)
	case path == "/food-log-groups" && r.Method == http.MethodGet:
		s.listFoodGroups(w, r, requestID, ctx)
	case strings.HasPrefix(path, "/food-log-groups/") && r.Method == http.MethodGet:
		s.getFoodGroup(w, r, requestID, ctx, strings.TrimPrefix(path, "/food-log-groups/"))
	case strings.HasPrefix(path, "/food-log-groups/") && r.Method == http.MethodDelete:
		s.voidFoodGroup(w, r, requestID, ctx, strings.TrimPrefix(path, "/food-log-groups/"))
	case strings.HasPrefix(path, "/food-logs/") && r.Method == http.MethodGet:
		s.getFoodLogItem(w, r, requestID, ctx, strings.TrimPrefix(path, "/food-logs/"))
	case strings.HasPrefix(path, "/food-logs/") && r.Method == http.MethodPatch:
		s.correctFoodLogItem(w, r, requestID, ctx, strings.TrimPrefix(path, "/food-logs/"))
	case strings.HasPrefix(path, "/food-logs/") && r.Method == http.MethodDelete:
		s.voidFoodLogItem(w, r, requestID, ctx, strings.TrimPrefix(path, "/food-logs/"))
	case path == "/dashboard" && r.Method == http.MethodGet:
		s.getDashboard(w, r, requestID, ctx)
	case path == "/history/nutrition" && r.Method == http.MethodGet:
		s.nutritionHistory(w, r, requestID, ctx)
	case path == "/nutrition/full-analysis" && r.Method == http.MethodGet:
		s.fullAnalysis(w, r, requestID, ctx)
	case path == "/history/health-items" && r.Method == http.MethodGet:
		s.healthActionHistory(w, r, requestID, ctx)
	case path == "/health-items" && r.Method == http.MethodGet:
		s.listHealthItems(w, r, requestID, ctx)
	case path == "/health-items" && r.Method == http.MethodPost:
		s.createHealthItem(w, r, requestID, ctx)
	case path == "/health-items/occurrences" && r.Method == http.MethodGet:
		s.listHealthOccurrences(w, r, requestID, ctx)
	case strings.HasPrefix(path, "/health-items/occurrences/") && r.Method == http.MethodPut:
		s.recordHealthOccurrence(w, r, requestID, ctx, strings.TrimPrefix(path, "/health-items/occurrences/"))
	case strings.HasPrefix(path, "/health-items/") && strings.HasSuffix(path, "/deactivate") && r.Method == http.MethodPost:
		s.deactivateHealthItem(w, r, requestID, ctx, strings.TrimSuffix(strings.TrimPrefix(path, "/health-items/"), "/deactivate"))
	case strings.HasPrefix(path, "/health-items/") && strings.HasSuffix(path, "/schedule-history") && r.Method == http.MethodGet:
		s.scheduleHistory(w, r, requestID, ctx, strings.TrimSuffix(strings.TrimPrefix(path, "/health-items/"), "/schedule-history"))
	case strings.HasPrefix(path, "/health-items/") && r.Method == http.MethodGet:
		s.getHealthItem(w, r, requestID, ctx, strings.TrimPrefix(path, "/health-items/"))
	case strings.HasPrefix(path, "/health-items/") && r.Method == http.MethodPatch:
		s.updateHealthItem(w, r, requestID, ctx, strings.TrimPrefix(path, "/health-items/"))
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

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
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
	case codes.ResourceExhausted:
		s.problem(w, requestID, 429, "USDA_SEARCH_LIMIT", "Your daily USDA search allowance is used. Search the local database or try again after midnight UTC.")
	case codes.Unavailable:
		switch status.Convert(err).Message() {
		case "USDA_DEMO_RATE_LIMITED":
			s.problem(w, requestID, 503, "USDA_PROVIDER_RATE_LIMITED", "USDA has temporarily rate-limited the shared demo connection. This is separate from your daily AteThat allowance. Search the local database or try USDA again later.")
		case "USDA_PROVIDER_RATE_LIMITED":
			s.problem(w, requestID, 503, "USDA_PROVIDER_RATE_LIMITED", "USDA has temporarily rate-limited the server connection. This is separate from your daily AteThat allowance. Search the local database or try USDA again later.")
		case "USDA_PROVIDER_AUTH_FAILED":
			s.problem(w, requestID, 503, "USDA_CONFIGURATION", "USDA rejected the server's API credentials. You can still search the local database.")
		case "USDA food provider unavailable", "USDA food provider returned invalid data":
			s.problem(w, requestID, 503, "USDA_UNAVAILABLE", "USDA search is temporarily unavailable. You can still search the local database or try again later.")
		default:
			s.problem(w, requestID, 503, "CORE_UNAVAILABLE", "Service temporarily unavailable")
		}
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
	case codes.AlreadyExists:
		s.problem(w, requestID, 409, "IDEMPOTENCY_KEY_REUSED", "Idempotency key was used for a different request")
	case codes.FailedPrecondition:
		message := strings.ToLower(status.Convert(err).Message())
		if strings.Contains(message, "compacted") || strings.Contains(message, "retention window") {
			s.problem(w, requestID, 409, "HISTORY_COMPACTED", "Detailed history is no longer available for this range; use complete daily summaries")
		} else if strings.Contains(message, "confirmed group") {
			s.problem(w, requestID, 409, "PRECONDITION_FAILED", "The food group is no longer available for correction")
		} else if strings.Contains(message, "outside-reference") {
			s.problem(w, requestID, 409, "TARGET_CONFIRMATION_REQUIRED", "Review and confirm the exact outside-reference targets before saving. If the review expired, review it again.")
		} else if strings.Contains(message, "targets") {
			s.problem(w, requestID, 409, "TARGET_REVIEW_REQUIRED", "Targets need a fresh review. Reload your current targets before trying again.")
		} else {
			s.problem(w, requestID, 410, "TOKEN_EXPIRED", "Token expired or already used")
		}
	default:
		s.problem(w, requestID, 503, "CORE_UNAVAILABLE", "Service temporarily unavailable")
	}
}
