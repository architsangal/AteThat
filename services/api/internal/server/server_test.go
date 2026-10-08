package server

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func signedToken(t *testing.T, key ed25519.PrivateKey, expires time.Time) string {
	t.Helper()
	header, _ := json.Marshal(map[string]string{"alg": "EdDSA", "kid": "local-v1"})
	claims, _ := json.Marshal(map[string]interface{}{
		"iss": "atethat-core", "aud": "atethat-api", "sub": "66f7e68012ab34cd56ef7890",
		"sid": "test-family",
		"iat": time.Now().Add(-time.Minute).Unix(), "exp": expires.Unix(),
	})
	message := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	return message + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, []byte(message)))
}

func TestVerifyAccessToken(t *testing.T) {
	key := ed25519.NewKeyFromSeed(make([]byte, 32))
	valid := signedToken(t, key, time.Now().Add(time.Minute))
	user, err := verifyAccessToken(valid, key.Public().(ed25519.PublicKey))
	if err != nil || user.UserID != "66f7e68012ab34cd56ef7890" || user.SessionFamily != "test-family" {
		t.Fatalf("valid token rejected: %v", err)
	}
	if _, err := verifyAccessToken(valid+"x", key.Public().(ed25519.PublicKey)); err == nil {
		t.Fatal("tampered token accepted")
	}
	if _, err := verifyAccessToken(signedToken(t, key, time.Now().Add(-time.Minute)), key.Public().(ed25519.PublicKey)); err == nil {
		t.Fatal("expired token accepted")
	}
}

func TestBasicHTTPBoundaries(t *testing.T) {
	s := &Server{Config: Config{FrontendOrigin: "http://localhost:5173"}}
	for _, test := range []struct {
		method, path, origin string
		expected             int
	}{
		{http.MethodGet, "/healthz", "", 200},
		{http.MethodGet, "/api/v1/does-not-exist", "", 404},
		{http.MethodOptions, "/api/v1/auth/login", "https://evil.example", 403},
		{http.MethodOptions, "/api/v1/auth/login", "http://localhost:5173", 204},
	} {
		req := httptest.NewRequest(test.method, test.path, nil)
		if test.origin != "" {
			req.Header.Set("Origin", test.origin)
		}
		response := httptest.NewRecorder()
		s.ServeHTTP(response, req)
		if response.Code != test.expected {
			t.Errorf("%s %s: got %d, want %d", test.method, test.path, response.Code, test.expected)
		}
		if response.Header().Get("X-Request-Id") == "" {
			t.Error("request ID missing")
		}
		if test.origin == "https://evil.example" && strings.Contains(response.Header().Get("Access-Control-Allow-Origin"), "evil") {
			t.Error("untrusted origin allowed")
		}
	}
}

func TestCompactedHistoryErrorIsNotTokenError(t *testing.T) {
	s := &Server{}
	w := httptest.NewRecorder()
	s.rpcProblem(w, "test-request", status.Error(codes.FailedPrecondition, "partial compacted day cannot be analyzed"))
	if w.Code != 409 || !strings.Contains(w.Body.String(), "HISTORY_COMPACTED") {
		t.Fatalf("wrong compacted-history response: %d %s", w.Code, w.Body.String())
	}
}
