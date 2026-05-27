package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/common"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/response"
)

func RateLimit(redisClient *redis.Client, prefix string, limit int64, window time.Duration, trustProxy bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if redisClient == nil || limit <= 0 || window <= 0 {
				next.ServeHTTP(w, r)
				return
			}
			key := fmt.Sprintf("rate:%s:%s", prefix, common.ClientIP(r, trustProxy))
			count, err := redisClient.Incr(r.Context(), key).Result()
			if err == nil && count == 1 {
				_ = redisClient.Expire(r.Context(), key, window).Err()
			}
			if err == nil && count > limit {
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", window.Seconds()))
				response.Error(w, r, http.StatusTooManyRequests, "rate_limited", "too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
