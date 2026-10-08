package server

import (
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	var limiter rateLimiter
	now := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		allowed, _ := limiter.allow("login|test-ip", 3, now)
		if !allowed {
			t.Fatalf("request %d denied early", i)
		}
	}
	if allowed, _ := limiter.allow("login|test-ip", 3, now); allowed {
		t.Fatal("limit bypassed")
	}
	if allowed, _ := limiter.allow("login|other-ip", 3, now); !allowed {
		t.Fatal("other client blocked")
	}
	if allowed, _ := limiter.allow("login|test-ip", 3, now.Add(time.Minute)); !allowed {
		t.Fatal("window did not reset")
	}
}
