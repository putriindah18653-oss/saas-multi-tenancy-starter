package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCleanRatePart(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"auth:login", "auth:login"},
		{"tenant:settings:update", "tenant:settings:update"},
		{"test.123", "test.123"},
		{"", "unknown"},
		{"hello-world", "hello-world"},
		{"UPPER-CASE", "upper-case"},
		{"spaced name", "spaced_name"},
		{"path/separator", "path_separator"},
	}
	for _, c := range cases {
		got := cleanRatePart(c.in)
		if got != c.want {
			t.Errorf("cleanRatePart(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestRateLimitIdentity(t *testing.T) {
	r, _ := http.NewRequest("GET", "/test", nil)
	r.RemoteAddr = "10.0.0.1:12345"
	if got := rateLimitIdentity(r, RateLimitByIP, false); got != "10.0.0.1" {
		t.Errorf("IP identity: got %q want 10.0.0.1", got)
	}
	if got := rateLimitIdentity(r, RateLimitByUser, false); got != "" {
		t.Errorf("User identity without auth context: got %q want empty", got)
	}
	if got := rateLimitIdentity(r, RateLimitByTenant, false); got != "" {
		t.Errorf("Tenant identity without tenant context: got %q want empty", got)
	}
}

func TestRateLimitNilRedisFailClosed(t *testing.T) {
	// When redisClient is nil AND active rules exist, the request should be blocked.
	handler := RateLimitRules(nil, false, RateLimitRule{Name: "test", Scope: RateLimitByIP, Limit: 10, Window: time.Minute})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when redis is nil with active rules, got %d", rec.Code)
	}
}

func TestRateLimitNilRedisFailOpenNoRules(t *testing.T) {
	// When redisClient is nil but there are no active rules (all Limit<=0),
	// the request should pass through.
	handler := RateLimitRules(nil, false, RateLimitRule{Name: "test", Scope: RateLimitByIP, Limit: 0, Window: time.Minute})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 when redis is nil with zero-limit rules (fail-open), got %d", rec.Code)
	}
}

func TestRateLimitNoActiveRules(t *testing.T) {
	// Zero-limit rules should pass through even with non-nil redis (which we can't test without real redis).
	// But with nil redis + all zero-limit rules, it MUST pass through.
	handler := RateLimitRules(nil, false,
		RateLimitRule{Name: "a", Scope: RateLimitByIP, Limit: 0, Window: time.Minute},
		RateLimitRule{Name: "b", Scope: RateLimitByIP, Limit: 0, Window: time.Minute},
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 with all zero-limit rules, got %d", rec.Code)
	}
}

func TestRateLimitBackwardCompat(t *testing.T) {
	handler := RateLimit(nil, "test", 10, time.Minute, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when redis is nil (backward compat), got %d", rec.Code)
	}
}
