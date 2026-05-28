package handler

import (
	"net/http"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/audit"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/common"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/response"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/middleware"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/tenant"
)

type AppSettingsHandler struct {
	svc        *tenant.Service
	audit      *audit.Service
	trustProxy bool
}

func NewAppSettingsHandler(s *tenant.Service, a *audit.Service, trustProxy bool) *AppSettingsHandler {
	return &AppSettingsHandler{svc: s, audit: a, trustProxy: trustProxy}
}
func (h *AppSettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.AppSettings(r.Context())
	if err != nil {
		response.Error(w, r, 500, "app_settings_failed", "could not load company settings")
		return
	}
	response.Success(w, r, 200, v)
}
func (h *AppSettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req tenant.AppSettings
	if !decode(w, r, &req) {
		return
	}
	v, err := h.svc.UpdateAppSettings(r.Context(), req)
	if err != nil {
		response.Error(w, r, 400, "app_settings_update_failed", "could not update company settings")
		return
	}
	if ac, ok := middleware.AuthFromContext(r.Context()); ok && h.audit != nil {
		_ = h.audit.Log(r.Context(), ac.UserID, "", "app.settings.update", "app_settings", "", map[string]any{}, common.ClientIP(r, h.trustProxy), r.UserAgent())
	}
	response.Success(w, r, 200, v)
}
