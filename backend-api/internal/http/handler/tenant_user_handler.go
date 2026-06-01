package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/audit"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/common"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/response"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/middleware"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/user"
)

type TenantUserHandler struct {
	svc        *user.Service
	audit      *audit.Service
	trustProxy bool
}

func NewTenantUserHandler(s *user.Service, a *audit.Service, trustProxy bool) *TenantUserHandler {
	return &TenantUserHandler{svc: s, audit: a, trustProxy: trustProxy}
}

type inviteReq struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}
type roleReq struct {
	Role string `json:"role"`
}

func (h *TenantUserHandler) Me(w http.ResponseWriter, r *http.Request) {
	ac, _ := middleware.AuthFromContext(r.Context())
	tc, _ := middleware.TenantFromContext(r.Context())
	m, err := h.svc.TenantMe(r.Context(), ac.UserID, tc.TenantID)
	if err != nil {
		response.Error(w, r, 404, "member_not_found", "member not found")
		return
	}
	response.Success(w, r, 200, m)
}
func (h *TenantUserHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	tc, _ := middleware.TenantFromContext(r.Context())
	response.Success(w, r, 200, map[string]any{"tenant_id": tc.TenantID, "status": "ok"})
}
func (h *TenantUserHandler) List(w http.ResponseWriter, r *http.Request) {
	tc, _ := middleware.TenantFromContext(r.Context())
	v, err := h.svc.List(r.Context(), tc.TenantID)
	if err != nil {
		response.Error(w, r, 500, "tenant_users_failed", "could not list tenant users")
		return
	}
	response.Success(w, r, 200, v)
}
func (h *TenantUserHandler) Invite(w http.ResponseWriter, r *http.Request) {
	var req inviteReq
	if !decode(w, r, &req) {
		return
	}
	if req.Role == "" {
		req.Role = "support"
	}
	tc, _ := middleware.TenantFromContext(r.Context())
	m, p, err := h.svc.Invite(r.Context(), tc.TenantID, req.Name, req.Email, req.Role)
	if err != nil {
		response.Error(w, r, 400, "invite_failed", "could not invite user")
		return
	}
	h.log(r, tc.TenantID, "tenant.user.invite", "user_tenant", m.ID)
	data := map[string]any{"member": m}
	if p != "" {
		data["temporary_password"] = p
	}
	response.Success(w, r, 201, data)
}
func (h *TenantUserHandler) ChangeRole(w http.ResponseWriter, r *http.Request) {
	var req roleReq
	if !decode(w, r, &req) {
		return
	}
	tc, _ := middleware.TenantFromContext(r.Context())
	id := chi.URLParam(r, "id")
	m, err := h.svc.ChangeRole(r.Context(), tc.TenantID, id, req.Role)
	if err != nil {
		response.Error(w, r, 400, "role_update_failed", "could not update role")
		return
	}
	h.log(r, tc.TenantID, "tenant.user.role_update", "user_tenant", id)
	response.Success(w, r, 200, m)
}
func (h *TenantUserHandler) Remove(w http.ResponseWriter, r *http.Request) {
	tc, _ := middleware.TenantFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.svc.Remove(r.Context(), tc.TenantID, id); err != nil {
		response.Error(w, r, 400, "remove_failed", "could not remove user")
		return
	}
	h.log(r, tc.TenantID, "tenant.user.remove", "user_tenant", id)
	response.Success(w, r, 200, map[string]any{"removed": true})
}
func (h *TenantUserHandler) log(r *http.Request, tenantID, action, typ, id string) {
	if h.audit == nil {
		return
	}
	ac, _ := middleware.AuthFromContext(r.Context())
	if err := h.audit.Log(r.Context(), ac.UserID, tenantID, action, typ, id, map[string]any{}, common.ClientIP(r, h.trustProxy), r.UserAgent()); err != nil {
		response.LogError(r, "audit write failed", "action", action, "error", err)
	}
}
