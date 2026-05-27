package middleware

import (
	"context"
	"net/http"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend/internal/http/response"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend/internal/rbac"
)

type tenantKey struct{}
type TenantContext struct{ TenantID, Role string }

func TenantFromContext(ctx context.Context) (TenantContext, bool) {
	v, ok := ctx.Value(tenantKey{}).(TenantContext)
	return v, ok
}
func RequireTenantAccess(s *rbac.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ac, ok := AuthFromContext(r.Context())
			if !ok {
				response.Error(w, r, 401, "unauthorized", "auth required")
				return
			}
			tid := r.Header.Get("X-Tenant-ID")
			if tid == "" {
				response.Error(w, r, 400, "tenant_required", "X-Tenant-ID header is required")
				return
			}
			role, member, err := s.TenantMembership(r.Context(), ac.UserID, tid)
			if err != nil || !member {
				response.Error(w, r, 403, "forbidden", "tenant access denied")
				return
			}
			tc := TenantContext{TenantID: tid, Role: role}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), tenantKey{}, tc)))
		})
	}
}
