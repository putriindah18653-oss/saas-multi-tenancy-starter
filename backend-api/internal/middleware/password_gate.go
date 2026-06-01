package middleware

import (
	"net/http"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/response"
)

// RequirePasswordChanged blocks authenticated routes for users that are still
// using an invited/temporary password. Reads MustChangePassword from the auth
// context (populated by RequireAuth) to avoid a redundant DB query.
func RequirePasswordChanged() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ac, ok := AuthFromContext(r.Context())
			if !ok {
				response.Error(w, r, http.StatusUnauthorized, "unauthorized", "auth required")
				return
			}
			if ac.MustChangePassword {
				response.Error(w, r, http.StatusForbidden, "password_change_required", "change password before accessing this resource")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
