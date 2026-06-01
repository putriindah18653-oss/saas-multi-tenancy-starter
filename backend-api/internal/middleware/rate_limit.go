package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/common"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/response"
)

// RateLimitScope names the entity whose request count is tracked.
//   - ip:     rate-limit by client IP (legacy default).
//   - user:   rate-limit by authenticated user ID.
//   - tenant: rate-limit by tenant ID (requires tenant context middleware).
type RateLimitScope string

const (
	RateLimitByIP     RateLimitScope = "ip"
	RateLimitByUser   RateLimitScope = "user"
	RateLimitByTenant RateLimitScope = "tenant"
)

// RateLimitRule defines one rate-limit constraint. Multiple rules can be
// combined with RateLimitRules to enforce, for example, per-IP AND per-user
// limits on the same endpoint.
//
// Redis keys follow the pattern:  rate:<name>:<scope>:<identity>
// where identity is derived from the request (ClientIP, UserID, or TenantID).
type RateLimitRule struct {
	Name   string
	Scope  RateLimitScope
	Limit  int64
	Window time.Duration
}

// RateLimit is the legacy single-rule shorthand. Prefer RateLimitRules for
// new endpoints so scope is explicit.
func RateLimit(redisClient *redis.Client, prefix string, limit int64, window time.Duration, trustProxy bool) func(http.Handler) http.Handler {
	return RateLimitRules(redisClient, trustProxy, RateLimitRule{Name: prefix, Scope: RateLimitByIP, Limit: limit, Window: window})
}

// RateLimitRules applies multiple rate-limit rules per request. Rules with
// Limit <= 0 or Window <= 0 are skipped (they disable rate limiting).
// When redisClient is nil or all rules are zero-valued the request passes
// through with no rate limiting (fail-open). Redis errors return 503.
func RateLimitRules(redisClient *redis.Client, trustProxy bool, rules ...RateLimitRule) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Collect active rules before the loop so we can decide fail-open.
			type activeRule struct {
				RateLimitRule
				identity string
			}
			var active []activeRule
			for _, rule := range rules {
				if rule.Limit <= 0 || rule.Window <= 0 {
					continue
				}
				identity := rateLimitIdentity(r, rule.Scope, trustProxy)
				if identity == "" {
					continue
				}
				active = append(active, activeRule{rule, identity})
			}
			if len(active) == 0 {
				// No active rules: fail-open.
				next.ServeHTTP(w, r)
				return
			}
			if redisClient == nil {
				// Redis unavailable but rules exist: fail-closed.
				response.Error(w, r, http.StatusServiceUnavailable, "service_unavailable", "rate limiter unavailable")
				return
			}
			for _, ar := range active {
				key := fmt.Sprintf("rate:%s:%s:%s", cleanRatePart(ar.Name), ar.Scope, cleanRatePart(ar.identity))
				count, err := redisClient.Incr(r.Context(), key).Result()
				if err != nil {
					slog.Default().Error("rate limit incr failed", "key", key, "error", err)
					response.Error(w, r, http.StatusServiceUnavailable, "service_unavailable", "rate limiter unavailable")
					return
				}
				// Always set expiry — idempotent and prevents permanent keys
				// when the initial EXPIRE is lost to a network blip.
				if err := redisClient.Expire(r.Context(), key, ar.Window).Err(); err != nil {
					slog.Default().Warn("rate limit expire failed", "key", key, "error", err)
				}
				if count > ar.Limit {
					w.Header().Set("Retry-After", fmt.Sprintf("%.0f", ar.Window.Seconds()))
					slog.Default().Warn("rate limited", "rule", ar.Name, "scope", string(ar.Scope))
					response.Error(w, r, http.StatusTooManyRequests, "rate_limited", "too many requests")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func rateLimitIdentity(r *http.Request, scope RateLimitScope, trustProxy bool) string {
	switch scope {
	case RateLimitByIP:
		return common.ClientIP(r, trustProxy)
	case RateLimitByUser:
		if ac, ok := AuthFromContext(r.Context()); ok {
			return ac.UserID
		}
	case RateLimitByTenant:
		if tc, ok := TenantFromContext(r.Context()); ok {
			return tc.TenantID
		}
	}
	return ""
}

// cleanRatePart normalises a rate-limit key segment to lowercase
// alphanumerics and a small set of separator characters. Invalid
// characters are replaced with underscores to prevent key collisions
// from accidental special characters.
func cleanRatePart(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	var b strings.Builder
	for _, r := range v {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == ':', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	if b.Len() == 0 {
		return "unknown"
	}
	return b.String()
}
