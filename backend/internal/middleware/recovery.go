package middleware

import (
	"log/slog"
	"net/http"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend/internal/http/response"
)

func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered", "panic", rec, "request_id", RequestIDFromContext(r.Context()))
					response.Error(w, r, http.StatusInternalServerError, "internal_error", "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
