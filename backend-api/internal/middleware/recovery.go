package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/response"
)

// Recovery catches panics in downstream handlers, logs the panic value and
// a full stack trace, and returns a 500 response. Stack traces are truncated
// to 4 KB to bound log volume.
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := debug.Stack()
					if len(stack) > 4096 {
						stack = stack[:4096]
					}
					logger.Error("panic recovered",
						"panic", rec,
						"request_id", RequestIDFromContext(r.Context()),
						"stack", string(stack),
					)
					response.Error(w, r, http.StatusInternalServerError, "internal_error", "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
