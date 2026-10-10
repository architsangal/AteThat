package security

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"
)

func TestPasswordHashAndVerify(t *testing.T) {
	one, err := PasswordHash("a safe test password")
	if err != nil {
		t.Fatal(err)
	}
	two, err := PasswordHash("a safe test password")
	if err != nil {
		t.Fatal(err)
	}
	if one == two || !VerifyPassword(one, "a safe test password") || VerifyPassword(one, "different") {
		t.Fatal("password hashing or verification failed")
	}
}

func TestTokens(t *testing.T) {
	first, err := RandomToken()
	if err != nil {
		t.Fatal(err)
	}
	second, err := RandomToken()
	if err != nil {
		t.Fatal(err)
	}
	if first == second || TokenHash(first) == first {
		t.Fatal("token generation or hashing failed")
	}
	seed := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		t.Fatal(err)
	}
	key, err := SigningKey(base64.StdEncoding.EncodeToString(seed))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AccessToken(key, "66f7e68012ab34cd56ef7890", "test-family", time.Now()); err != nil {
		t.Fatal(err)
	}
}
