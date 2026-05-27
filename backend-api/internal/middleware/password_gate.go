package middleware

import (
	"net/http"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/auth"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/response"
)

// RequirePasswordChanged blocks authenticated routes for users that are still
// using an invited/temporary password. Keep only the minimum self-service auth
// routes (me, logout, change password) outside this middleware.
func RequirePasswordChanged(s *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ac, ok := AuthFromContext(r.Context())
			if !ok {
				response.Error(w, r, http.StatusUnauthorized, "unauthorized", "auth required")
				return
			}
			u, err := s.Me(r.Context(), ac.UserID)
			if err != nil {
				response.Error(w, r, http.StatusUnauthorized, "unauthorized", "user not found")
				return
			}
			if u.MustChangePassword {
				response.Error(w, r, http.StatusForbidden, "password_change_required", "change password before accessing this resource")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
