package server

import (
	"net/http/httptest"
	"net/netip"
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

func TestClientIPOnlyTrustsConfiguredProxy(t *testing.T) {
	trusted := netip.MustParsePrefix("172.18.0.0/16")
	server := &Server{Config: Config{TrustedProxyCIDRs: []netip.Prefix{trusted}}}
	request := httptest.NewRequest("POST", "http://api.test/api/v1/auth/login", nil)
	request.RemoteAddr = "172.18.0.4:51321"
	request.Header.Set("X-Forwarded-For", "198.51.100.9, 203.0.113.18")
	if got := server.clientIP(request); got != "203.0.113.18" {
		t.Fatalf("trusted proxy client IP = %q", got)
	}
	request.RemoteAddr = "198.51.100.5:51321"
	if got := server.clientIP(request); got != "198.51.100.5" {
		t.Fatalf("untrusted peer must not control XFF, got %q", got)
	}
}
