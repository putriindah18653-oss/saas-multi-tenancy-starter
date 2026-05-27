package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/audit"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/response"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/middleware"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/tenant"
)

type TenantHandler struct {
	svc   *tenant.Service
	audit *audit.Service
}

func NewTenantHandler(s *tenant.Service, a *audit.Service) *TenantHandler {
	return &TenantHandler{svc: s, audit: a}
}

type tenantReq struct {
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Status string `json:"status"`
}

func (h *TenantHandler) List(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.List(r.Context())
	if err != nil {
		response.Error(w, r, 500, "tenant_list_failed", "could not list tenants")
		return
	}
	response.Success(w, r, 200, v)
}
func (h *TenantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req tenantReq
	if !decode(w, r, &req) {
		return
	}
	ac, _ := middleware.AuthFromContext(r.Context())
	t, err := h.svc.Create(r.Context(), ac.UserID, req.Name, req.Slug)
	if err != nil {
		response.Error(w, r, 400, "tenant_create_failed", "could not create tenant")
		return
	}
	h.log(r, "tenant.create", t.ID)
	response.Success(w, r, 201, t)
}
func (h *TenantHandler) Get(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, 404, "tenant_not_found", "tenant not found")
		return
	}
	response.Success(w, r, 200, t)
}
func (h *TenantHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req tenantReq
	if !decode(w, r, &req) {
		return
	}
	id := chi.URLParam(r, "id")
	t, err := h.svc.Update(r.Context(), id, req.Name, req.Status)
	if err != nil {
		response.Error(w, r, 400, "tenant_update_failed", "could not update tenant")
		return
	}
	h.log(r, "tenant.update", id)
	response.Success(w, r, 200, t)
}
func (h *TenantHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		response.Error(w, r, 404, "tenant_not_found", "tenant not found")
		return
	}
	h.log(r, "tenant.delete", id)
	response.Success(w, r, 200, map[string]any{"deleted": true})
}
func (h *TenantHandler) log(r *http.Request, action, id string) {
	ac, _ := middleware.AuthFromContext(r.Context())
	_ = h.audit.Log(r.Context(), ac.UserID, "", action, "tenant", id, map[string]any{}, r.RemoteAddr, r.UserAgent())
}
