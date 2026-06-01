package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/auth"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/response"
)

type authKey struct{}

func AuthFromContext(ctx context.Context) (auth.Context, bool) {
	v, ok := ctx.Value(authKey{}).(auth.Context)
	return v, ok
}

// RequireAuth parses the Bearer token, fetches the user profile, and stores
// auth.Context (including MustChangePassword) in the request context.
func RequireAuth(s *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				response.Error(w, r, http.StatusUnauthorized, "unauthorized", "missing bearer token")
				return
			}
			c, err := s.Parse(strings.TrimPrefix(h, "Bearer "), "access")
			if err != nil {
				response.Error(w, r, http.StatusUnauthorized, "unauthorized", "invalid token")
				return
			}
			u, err := s.Me(r.Context(), c.UserID)
			if err != nil || !u.IsActive {
				response.Error(w, r, http.StatusUnauthorized, "unauthorized", "user inactive")
				return
			}
			ac := auth.Context{
				UserID:            u.ID,
				Email:             u.Email,
				AppRole:           u.AppRole,
				MustChangePassword: u.MustChangePassword,
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authKey{}, ac)))
		})
	}
}
