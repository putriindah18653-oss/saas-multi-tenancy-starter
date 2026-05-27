package middleware

import (
	"net/http"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/response"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/rbac"
)

func RequireAppRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ac, ok := AuthFromContext(r.Context())
			if !ok {
				response.Error(w, r, 401, "unauthorized", "auth required")
				return
			}
			if ac.AppRole != "owner-app" && ac.AppRole != role {
				response.Error(w, r, 403, "forbidden", "insufficient app role")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
func RequireTenantRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tc, ok := TenantFromContext(r.Context())
			if !ok {
				response.Error(w, r, 403, "forbidden", "tenant context required")
				return
			}
			if tc.Role != "owner-tenant" && tc.Role != role {
				response.Error(w, r, 403, "forbidden", "insufficient tenant role")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
func RequirePermission(s *rbac.Service, key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ac, ok := AuthFromContext(r.Context())
			if !ok {
				response.Error(w, r, 401, "unauthorized", "auth required")
				return
			}
			tc, tenantOK := TenantFromContext(r.Context())
			if len(key) > 7 && key[:7] == "tenant." && !tenantOK {
				response.Error(w, r, 403, "forbidden", "tenant context required")
				return
			}
			allowed, err := s.HasPermission(r.Context(), ac.AppRole, tc.Role, key)
			if err != nil {
				response.Error(w, r, 500, "permission_check_failed", "could not check permission")
				return
			}
			if !allowed {
				response.Error(w, r, 403, "forbidden", "missing permission")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
