package handler

import (
	"net/http"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/auth"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/response"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/middleware"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/rbac"
)

type MeHandler struct {
	auth *auth.Service
	rbac *rbac.Service
}

func NewMeHandler(a *auth.Service, r *rbac.Service) *MeHandler { return &MeHandler{auth: a, rbac: r} }
func (h *MeHandler) Permissions(w http.ResponseWriter, r *http.Request) {
	ac, _ := middleware.AuthFromContext(r.Context())
	app, _ := h.rbac.AppPermissions(r.Context(), ac.AppRole)
	tc, _ := middleware.TenantFromContext(r.Context())
	ten := []string{}
	if tc.Role != "" {
		ten, _ = h.rbac.TenantPermissions(r.Context(), tc.Role)
	}
	response.Success(w, r, 200, map[string]any{"app_role": ac.AppRole, "tenant_role": tc.Role, "app_permissions": app, "tenant_permissions": ten})
}
func (h *MeHandler) Tenants(w http.ResponseWriter, r *http.Request) {
	ac, _ := middleware.AuthFromContext(r.Context())
	m, err := h.auth.Memberships(r.Context(), ac.UserID)
	if err != nil {
		response.Error(w, r, 500, "tenant_lookup_failed", "could not load tenants")
		return
	}
	response.Success(w, r, 200, m)
}
