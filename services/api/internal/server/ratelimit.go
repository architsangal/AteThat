package server

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type rateWindow struct {
	Count   int
	ResetAt time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	windows map[string]rateWindow
}

func (l *rateLimiter) allow(key string, limit int, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.windows == nil {
		l.windows = make(map[string]rateWindow)
	}
	if len(l.windows) > 10000 {
		for key, value := range l.windows {
			if !now.Before(value.ResetAt) {
				delete(l.windows, key)
			}
		}
	}
	window := l.windows[key]
	if !now.Before(window.ResetAt) {
		window = rateWindow{ResetAt: now.Add(time.Minute)}
	}
	if window.Count >= limit {
		return false, window.ResetAt.Sub(now)
	}
	window.Count++
	l.windows[key] = window
	return true, 0
}

func authLimit(path string) int {
	switch path {
	case "/api/v1/auth/login":
		return 10
	case "/api/v1/auth/register", "/api/v1/auth/resend-verification", "/api/v1/auth/forgot-password", "/api/v1/auth/cancel-deletion":
		return 5
	case "/api/v1/auth/verify-email", "/api/v1/auth/reset-password":
		return 10
	default:
		return 0
	}
}

func (s *Server) limitPublicAuth(w http.ResponseWriter, r *http.Request, requestID string) bool {
	if r.Method != http.MethodPost {
		return true
	}
	limit := authLimit(r.URL.Path)
	if limit == 0 {
		return true
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	if ip == "" {
		ip = "unknown"
	}
	allowed, retry := s.authLimiter.allow(r.URL.Path+"|"+ip, limit, time.Now())
	if allowed {
		return true
	}
	seconds := int(retry.Seconds()) + 1
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	s.problem(w, requestID, http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests; please retry later")
	return false
}
