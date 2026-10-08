package server

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"github.com/architsangal/AteThat/services/core/internal/security"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const googleAuthorizationURL = "https://accounts.google.com/o/oauth2/v2/auth"
const googleTokenURL = "https://oauth2.googleapis.com/token"
const googleJWKSURL = "https://www.googleapis.com/oauth2/v3/certs"

func (s *Server) googleConfigured() bool {
	return s.Cfg.GoogleClientID != "" && s.Cfg.GoogleClientSecret != "" && s.Cfg.GoogleRedirectURI != ""
}

func allowedReturnTo(path string) bool {
	return path == "/app" || path == "/app/settings" || path == "/cancel-deletion"
}

func (s *Server) StartGoogleSignIn(ctx context.Context, req *pb.StartGoogleSignInRequest) (*pb.StartGoogleSignInResponse, error) {
	return s.startGoogle(ctx, "SIGN_IN", nil, req.ReturnTo)
}

func (s *Server) StartGoogleLink(ctx context.Context, _ *pb.Empty) (*pb.StartGoogleSignInResponse, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	return s.startGoogle(ctx, "LINK", &userID, "/app/settings")
}

func (s *Server) StartGoogleDeletionRequest(ctx context.Context, _ *pb.Empty) (*pb.StartGoogleSignInResponse, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	return s.startGoogle(ctx, "REQUEST_DELETION", &userID, "/cancel-deletion")
}

// StartGoogleDeletionCancellation intentionally has no authenticated principal:
// a deletion request revokes every session. A fresh Google consent flow proves
// control of the linked Google identity before restoring the pending account.
func (s *Server) StartGoogleDeletionCancellation(ctx context.Context, _ *pb.Empty) (*pb.StartGoogleSignInResponse, error) {
	return s.startGoogle(ctx, "CANCEL_DELETION", nil, "/app/settings")
}

func (s *Server) startGoogle(ctx context.Context, operation string, userID *primitive.ObjectID, returnTo string) (*pb.StartGoogleSignInResponse, error) {
	if !s.googleConfigured() {
		return nil, status.Error(codes.Unavailable, "Google sign-in is not configured")
	}
	if !allowedReturnTo(returnTo) {
		return nil, invalid("invalid return path")
	}
	state, err := security.RandomToken()
	if err != nil {
		return nil, internal()
	}
	nonce, err := security.RandomToken()
	if err != nil {
		return nil, internal()
	}
	verifier, err := security.RandomToken()
	if err != nil {
		return nil, internal()
	}
	now := time.Now().UTC()
	doc := oauthStateDoc{ID: primitive.NewObjectID(), StateHash: security.TokenHash(state), Nonce: nonce,
		CodeVerifier: verifier, ReturnTo: returnTo, Operation: operation, UserID: userID,
		ExpiresAt: now.Add(10 * time.Minute), CreatedAt: now}
	if _, err := s.DB.Collection("oauth_states").InsertOne(ctx, doc); err != nil {
		return nil, internal()
	}
	challenge := sha256.Sum256([]byte(verifier))
	query := url.Values{
		"client_id": {s.Cfg.GoogleClientID}, "redirect_uri": {s.Cfg.GoogleRedirectURI},
		"response_type": {"code"}, "scope": {"openid email profile"}, "state": {state}, "nonce": {nonce},
		"code_challenge": {base64.RawURLEncoding.EncodeToString(challenge[:])}, "code_challenge_method": {"S256"},
	}
	return &pb.StartGoogleSignInResponse{AuthorizationUrl: googleAuthorizationURL + "?" + query.Encode()}, nil
}

func (s *Server) CompleteGoogleSignIn(ctx context.Context, req *pb.CompleteGoogleSignInRequest) (*pb.CompleteGoogleSignInResponse, error) {
	if !s.googleConfigured() {
		return nil, status.Error(codes.Unavailable, "Google sign-in is not configured")
	}
	if req.Code == "" || req.State == "" {
		return nil, invalid("missing OAuth callback parameters")
	}
	now := time.Now().UTC()
	var state oauthStateDoc
	filter := bson.M{"state_hash": security.TokenHash(req.State), "consumed_at": nil, "expires_at": bson.M{"$gt": now}}
	err := s.DB.Collection("oauth_states").FindOneAndUpdate(ctx, filter, bson.M{"$set": bson.M{"consumed_at": now}},
		options.FindOneAndUpdate().SetReturnDocument(options.Before)).Decode(&state)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "OAuth state expired or already used")
	}
	idToken, err := s.exchangeGoogleCode(ctx, req.Code, state.CodeVerifier)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Google sign-in failed")
	}
	claims, err := s.verifyGoogleIDToken(ctx, idToken, state.Nonce)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Google sign-in failed")
	}
	var user userDoc
	var created bool
	if state.Operation == "LINK" && state.UserID != nil {
		user, err = s.linkGoogleUser(ctx, *state.UserID, claims)
	} else if state.Operation == "REQUEST_DELETION" && state.UserID != nil {
		user, err = s.requestGoogleDeletion(ctx, *state.UserID, claims)
		if err == nil {
			return &pb.CompleteGoogleSignInResponse{ReturnTo: "/cancel-deletion"}, nil
		}
	} else if state.Operation == "CANCEL_DELETION" {
		user, err = s.cancelGoogleDeletion(ctx, claims)
	} else if state.Operation == "SIGN_IN" {
		user, created, err = s.findOrCreateGoogleUser(ctx, claims)
	} else {
		return nil, status.Error(codes.FailedPrecondition, "invalid OAuth state")
	}
	if err != nil {
		return nil, err
	}
	session, err := s.newSession(ctx, user)
	if err != nil {
		return nil, err
	}
	return &pb.CompleteGoogleSignInResponse{Session: session, ReturnTo: state.ReturnTo, NewUser: created}, nil
}

func (s *Server) requestGoogleDeletion(ctx context.Context, userID primitive.ObjectID, claims googleClaims) (userDoc, error) {
	var identity identityDoc
	if err := s.DB.Collection("auth_identities").FindOne(ctx, bson.M{"user_id": userID, "provider": "GOOGLE", "subject": claims.Subject}).Decode(&identity); err != nil {
		return userDoc{}, status.Error(codes.Unauthenticated, "Google identity is not linked to this account")
	}
	var user userDoc
	if err := s.DB.Collection("users").FindOne(ctx, bson.M{"_id": userID, "status": "ACTIVE"}).Decode(&user); err != nil {
		return userDoc{}, status.Error(codes.FailedPrecondition, "account is unavailable for deletion")
	}
	_, err := s.requestDeletionForUser(ctx, userID)
	return user, err
}

func (s *Server) cancelGoogleDeletion(ctx context.Context, claims googleClaims) (userDoc, error) {
	var identity identityDoc
	if err := s.DB.Collection("auth_identities").FindOne(ctx, bson.M{"provider": "GOOGLE", "subject": claims.Subject}).Decode(&identity); err != nil {
		return userDoc{}, status.Error(codes.Unauthenticated, "Google account cannot cancel deletion")
	}
	var user userDoc
	if err := s.DB.Collection("users").FindOne(ctx, bson.M{"_id": identity.UserID, "status": "DELETION_PENDING"}).Decode(&user); err != nil {
		return userDoc{}, status.Error(codes.FailedPrecondition, "account deletion is not pending")
	}
	if err := s.cancelDeletionForUser(ctx, user.ID); err != nil {
		return userDoc{}, err
	}
	user.Status = "ACTIVE"
	return user, nil
}

func (s *Server) linkGoogleUser(ctx context.Context, userID primitive.ObjectID, claims googleClaims) (userDoc, error) {
	var user userDoc
	if err := s.DB.Collection("users").FindOne(ctx, bson.M{"_id": userID, "status": "ACTIVE"}).Decode(&user); err != nil {
		return user, status.Error(codes.PermissionDenied, "account unavailable")
	}
	var existing identityDoc
	err := s.DB.Collection("auth_identities").FindOne(ctx, bson.M{"provider": "GOOGLE", "subject": claims.Subject}).Decode(&existing)
	if err == nil {
		if existing.UserID == userID {
			return user, nil
		}
		return userDoc{}, status.Error(codes.AlreadyExists, "Google identity already linked to another account")
	}
	if err != mongo.ErrNoDocuments {
		return userDoc{}, internal()
	}
	now := time.Now().UTC()
	identity := identityDoc{ID: primitive.NewObjectID(), UserID: userID, IdentityType: "GOOGLE", Provider: "GOOGLE",
		Subject: claims.Subject, SchemaVersion: 1, CreatedAt: now, UpdatedAt: now}
	if _, err := s.DB.Collection("auth_identities").InsertOne(ctx, identity); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return userDoc{}, status.Error(codes.AlreadyExists, "Google identity already linked")
		}
		return userDoc{}, internal()
	}
	return user, nil
}

func (s *Server) exchangeGoogleCode(ctx context.Context, code, verifier string) (string, error) {
	form := url.Values{"client_id": {s.Cfg.GoogleClientID}, "client_secret": {s.Cfg.GoogleClientSecret},
		"code": {code}, "code_verifier": {verifier}, "grant_type": {"authorization_code"}, "redirect_uri": {s.Cfg.GoogleRedirectURI}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", errors.New("Google token exchange rejected")
	}
	var result struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil || result.IDToken == "" {
		return "", errors.New("missing ID token")
	}
	return result.IDToken, nil
}

type googleClaims struct {
	Issuer        string `json:"iss"`
	Audience      string `json:"aud"`
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Nonce         string `json:"nonce"`
	ExpiresAt     int64  `json:"exp"`
	IssuedAt      int64  `json:"iat"`
}

func (s *Server) verifyGoogleIDToken(ctx context.Context, token, expectedNonce string) (googleClaims, error) {
	var claims googleClaims
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return claims, errors.New("invalid ID token")
	}
	headerRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return claims, err
	}
	var header struct {
		Algorithm string `json:"alg"`
		KeyID     string `json:"kid"`
	}
	if err := json.Unmarshal(headerRaw, &header); err != nil || header.Algorithm != "RS256" || header.KeyID == "" {
		return claims, errors.New("unsupported ID token algorithm")
	}
	key, err := googlePublicKey(ctx, header.KeyID)
	if err != nil {
		return claims, err
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return claims, err
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature); err != nil {
		return claims, err
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || json.Unmarshal(payload, &claims) != nil {
		return claims, errors.New("invalid ID token claims")
	}
	now := time.Now().Unix()
	_, emailOK := validEmail(claims.Email)
	if (claims.Issuer != "https://accounts.google.com" && claims.Issuer != "accounts.google.com") ||
		claims.Audience != s.Cfg.GoogleClientID || claims.Subject == "" || !claims.EmailVerified || !emailOK ||
		claims.Nonce != expectedNonce || claims.ExpiresAt <= now || claims.IssuedAt > now+30 || claims.IssuedAt < now-3600 {
		return claims, errors.New("ID token claims rejected")
	}
	return claims, nil
}

func googlePublicKey(ctx context.Context, keyID string) (*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleJWKSURL, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("Google keys unavailable")
	}
	var keys struct {
		Keys []struct {
			KeyID     string `json:"kid"`
			Type      string `json:"kty"`
			Algorithm string `json:"alg"`
			Use       string `json:"use"`
			N         string `json:"n"`
			E         string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&keys); err != nil {
		return nil, err
	}
	for _, key := range keys.Keys {
		if key.KeyID != keyID || key.Type != "RSA" || key.Algorithm != "RS256" || key.Use != "sig" {
			continue
		}
		nBytes, nErr := base64.RawURLEncoding.DecodeString(key.N)
		eBytes, eErr := base64.RawURLEncoding.DecodeString(key.E)
		if nErr != nil || eErr != nil || len(nBytes) < 256 || len(eBytes) < 1 || len(eBytes) > 4 {
			return nil, errors.New("invalid Google key")
		}
		exponent := 0
		for _, value := range eBytes {
			exponent = exponent<<8 | int(value)
		}
		if exponent < 3 {
			return nil, errors.New("invalid Google exponent")
		}
		return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: exponent}, nil
	}
	return nil, errors.New("Google key ID not found")
}

func (s *Server) findOrCreateGoogleUser(ctx context.Context, claims googleClaims) (userDoc, bool, error) {
	var identity identityDoc
	err := s.DB.Collection("auth_identities").FindOne(ctx, bson.M{"provider": "GOOGLE", "subject": claims.Subject}).Decode(&identity)
	if err == nil {
		var user userDoc
		if err := s.DB.Collection("users").FindOne(ctx, bson.M{"_id": identity.UserID, "status": "ACTIVE"}).Decode(&user); err != nil {
			return user, false, status.Error(codes.PermissionDenied, "account unavailable")
		}
		return user, false, nil
	}
	if err != mongo.ErrNoDocuments {
		return userDoc{}, false, internal()
	}
	email, ok := validEmail(claims.Email)
	if !ok {
		return userDoc{}, false, status.Error(codes.Unauthenticated, "Google email invalid")
	}
	var existing userDoc
	if err := s.DB.Collection("users").FindOne(ctx, bson.M{"email_normalized": email}).Decode(&existing); err == nil {
		return userDoc{}, false, status.Error(codes.AlreadyExists, "email already has an AteThat account; explicit linking is required")
	} else if err != mongo.ErrNoDocuments {
		return userDoc{}, false, internal()
	}
	now := time.Now().UTC()
	name := strings.TrimSpace(claims.Name)
	if name == "" || len(name) > 80 {
		name = strings.SplitN(email, "@", 2)[0]
	}
	user := userDoc{ID: primitive.NewObjectID(), EmailNormalized: email, EmailDisplay: claims.Email, EmailVerifiedAt: &now,
		Status: "ACTIVE", SchemaVersion: 1, Version: 1, CreatedAt: now, UpdatedAt: now}
	newIdentity := identityDoc{ID: primitive.NewObjectID(), UserID: user.ID, IdentityType: "GOOGLE", Provider: "GOOGLE",
		Subject: claims.Subject, SchemaVersion: 1, CreatedAt: now, UpdatedAt: now}
	profile := profileDoc{ID: primitive.NewObjectID(), UserID: user.ID, DisplayName: name, Timezone: "UTC",
		SchemaVersion: 1, Version: 1, CreatedAt: now, UpdatedAt: now}
	preferences := preferencesDoc{ID: primitive.NewObjectID(), UserID: user.ID, DailyResetTime: "04:00", UnitSystem: "METRIC",
		DashboardNutrients: []string{"ENERGY", "PROTEIN", "CARBOHYDRATE", "FAT", "FIBER"},
		SchemaVersion:      1, Version: 1, CreatedAt: now, UpdatedAt: now}
	session, err := s.Client.StartSession()
	if err != nil {
		return userDoc{}, false, internal()
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		for collection, doc := range map[string]interface{}{"users": user, "auth_identities": newIdentity,
			"user_profiles": profile, "user_preferences": preferences} {
			if _, err := s.DB.Collection(collection).InsertOne(sc, doc); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	if mongo.IsDuplicateKeyError(err) {
		return userDoc{}, false, status.Error(codes.AlreadyExists, "account already exists")
	}
	if err != nil {
		return userDoc{}, false, internal()
	}
	return user, true, nil
}
