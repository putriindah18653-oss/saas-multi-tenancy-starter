package handler

import (
	"net/http"
	"strconv"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/audit"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/response"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/middleware"
)

type AuditHandler struct{ svc *audit.Service }

func NewAuditHandler(s *audit.Service) *AuditHandler { return &AuditHandler{svc: s} }
func (h *AuditHandler) AppList(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.ListApp(r.Context(), parseLimit(r))
	if err != nil {
		response.LogError(r, "list app audit failed", "error", err)
		response.LogError(r, "list tenant audit failed", "error", err)
		response.Error(w, r, 500, "audit_failed", "could not list audit logs")
		return
	}
	response.Success(w, r, 200, v)
}
func (h *AuditHandler) TenantList(w http.ResponseWriter, r *http.Request) {
	tc, _ := middleware.TenantFromContext(r.Context())
	v, err := h.svc.ListTenant(r.Context(), tc.TenantID, parseLimit(r))
	if err != nil {
		response.Error(w, r, 500, "audit_failed", "could not list audit logs")
		return
	}
	response.Success(w, r, 200, v)
}
func parseLimit(r *http.Request) int { n, _ := strconv.Atoi(r.URL.Query().Get("limit")); return n }
