package handler

import (
	"net/http"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/audit"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/common"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/response"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/middleware"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/tenant"
)

type TenantSettingsHandler struct {
	svc        *tenant.Service
	audit      *audit.Service
	trustProxy bool
}

func NewTenantSettingsHandler(s *tenant.Service, a *audit.Service, trustProxy bool) *TenantSettingsHandler {
	return &TenantSettingsHandler{svc: s, audit: a, trustProxy: trustProxy}
}
func (h *TenantSettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	tc, _ := middleware.TenantFromContext(r.Context())
	v, err := h.svc.Settings(r.Context(), tc.TenantID)
	if err != nil {
		response.Error(w, r, 500, "settings_failed", "could not load tenant settings")
		return
	}
	response.Success(w, r, 200, v)
}
func (h *TenantSettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req tenant.Settings
	if !decode(w, r, &req) {
		return
	}
	tc, _ := middleware.TenantFromContext(r.Context())
	v, err := h.svc.UpdateSettings(r.Context(), tc.TenantID, req)
	if err != nil {
		response.Error(w, r, 400, "settings_update_failed", "could not update tenant settings")
		return
	}
	if ac, ok := middleware.AuthFromContext(r.Context()); ok && h.audit != nil {
		if err := h.audit.Log(r.Context(), ac.UserID, tc.TenantID, "tenant.settings.update", "tenant_settings", tc.TenantID, map[string]any{}, common.ClientIP(r, h.trustProxy), r.UserAgent()); err != nil {
			response.LogError(r, "audit write failed", "action", "tenant.settings.update", "error", err)
		}
	}
	response.Success(w, r, 200, v)
}
