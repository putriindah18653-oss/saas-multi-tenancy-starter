package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/response"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/rbac"
)

type tenantKey struct{}
type TenantContext struct {
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
	Source   string `json:"source"`
}

func TenantFromContext(ctx context.Context) (TenantContext, bool) {
	v, ok := ctx.Value(tenantKey{}).(TenantContext)
	return v, ok
}

// RequireTenantAccess resolves the target tenant from headers and verifies
// the authenticated user is an active member.
//
// Header precedence:
//  1. X-Internal-Tenant-ID — trusted only from the Dashboard Worker. The
//     worker MUST send X-Internal-Proxy-Secret matching the server's
//     INTERNAL_PROXY_SECRET configuration. Without a valid secret the
//     header is treated as untrusted and skipped.
//  2. X-Tenant-ID — hint from client frontend, cross-checked against membership.
//
// Error responses distinguish infrastructure failures from authorization
// failures so that load balancers do not retry auth decisions.
func RequireTenantAccess(s *rbac.Service, internalProxySecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ac, ok := AuthFromContext(r.Context())
			if !ok {
				response.Error(w, r, 401, "unauthorized", "auth required")
				return
			}
			tid := r.Header.Get("X-Internal-Tenant-ID")
			source := "internal_proxy"
			// Verify that the caller is the trusted Dashboard Worker. When
			// INTERNAL_PROXY_SECRET is unset (dev), internal-proxy headers are
			// still accepted — production MUST set a strong shared secret.
			if tid != "" {
				if internalProxySecret != "" && r.Header.Get("X-Internal-Proxy-Secret") != internalProxySecret {
					// Treat spoofed internal header as untrusted; fall back to client header.
					tid = r.Header.Get("X-Tenant-ID")
					source = "header"
				}
			}
			if tid == "" {
				tid = r.Header.Get("X-Tenant-ID")
				source = "header"
			}
			if tid == "" {
				response.Error(w, r, 400, "tenant_required", "X-Tenant-ID header is required")
				return
			}
			checkCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			role, member, err := s.TenantMembership(checkCtx, ac.UserID, tid)
			if err != nil {
				// Infrastructure error (DB down, timeout) — do not treat as auth failure.
				response.Error(w, r, 500, "tenant_check_error", "could not check tenant access")
				return
			}
			if !member {
				response.Error(w, r, 403, "tenant_access_denied", "tenant access denied")
				return
			}
			tc := TenantContext{TenantID: tid, Role: role, Source: source}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), tenantKey{}, tc)))
		})
	}
}
