package handler

import (
	"net/http"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend/internal/auth"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend/internal/http/response"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend/internal/middleware"
)

type AuthHandler struct{ svc *auth.Service }

func NewAuthHandler(s *auth.Service) *AuthHandler { return &AuthHandler{svc: s} }

type registerReq struct{ Name, Email, Password string }
type loginReq struct{ Email, Password string }
type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) RegisterOwner(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if !decode(w, r, &req) {
		return
	}
	if req.Email == "" || req.Password == "" || req.Name == "" {
		response.Error(w, r, 400, "validation_error", "name, email, and password are required")
		return
	}
	u, a, rt, err := h.svc.RegisterOwner(r.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		response.Error(w, r, 400, "register_owner_failed", err.Error())
		return
	}
	response.Success(w, r, 201, map[string]any{"user": u, "access_token": a, "refresh_token": rt})
}
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if !decode(w, r, &req) {
		return
	}
	u, a, rt, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		response.Error(w, r, 401, "invalid_credentials", "invalid email or password")
		return
	}
	response.Success(w, r, 200, map[string]any{"user": u, "access_token": a, "refresh_token": rt})
}
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if !decode(w, r, &req) {
		return
	}
	c, err := h.svc.Parse(req.RefreshToken, "refresh")
	if err != nil {
		response.Error(w, r, 401, "invalid_refresh_token", "invalid refresh token")
		return
	}
	u, err := h.svc.Me(r.Context(), c.UserID)
	if err != nil {
		response.Error(w, r, 401, "invalid_refresh_token", "user not found")
		return
	}
	a, rt, err := h.svc.Tokens(u)
	if err != nil {
		response.Error(w, r, 500, "token_error", "could not create token")
		return
	}
	response.Success(w, r, 200, map[string]any{"access_token": a, "refresh_token": rt})
}
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	response.Success(w, r, 200, map[string]any{"status": "logged_out"})
}
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	ac, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		response.Error(w, r, 401, "unauthorized", "missing auth context")
		return
	}
	u, err := h.svc.Me(r.Context(), ac.UserID)
	if err != nil {
		response.Error(w, r, 404, "not_found", "user not found")
		return
	}
	response.Success(w, r, 200, u)
}
