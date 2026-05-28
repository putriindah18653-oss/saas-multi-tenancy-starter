package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/audit"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/auth"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/common"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/response"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/middleware"
)

type AuthHandler struct {
	svc        *auth.Service
	audit      *audit.Service
	trustProxy bool
}

func NewAuthHandler(s *auth.Service, a *audit.Service, trustProxy bool) *AuthHandler {
	return &AuthHandler{svc: s, audit: a, trustProxy: trustProxy}
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}
type logoutReq struct {
	RefreshToken string `json:"refresh_token"`
}
type changePasswordReq struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if !decode(w, r, &req) {
		return
	}
	u, a, rt, err := h.svc.Login(r.Context(), req.Email, req.Password, h.clientIP(r), r.UserAgent())
	if err != nil {
		response.Error(w, r, 401, "invalid_credentials", "invalid email or password")
		return
	}
	h.log(r, u.ID, "auth.login", "user", u.ID)
	response.Success(w, r, 200, map[string]any{"user": u, "tenant_memberships": u.Tenants, "access_token": a, "refresh_token": rt})
}
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if !decode(w, r, &req) {
		return
	}
	u, a, rt, err := h.svc.Refresh(r.Context(), req.RefreshToken, h.clientIP(r), r.UserAgent())
	if err != nil {
		response.Error(w, r, 401, "invalid_refresh_token", "invalid refresh token")
		return
	}
	response.Success(w, r, 200, map[string]any{"user": u, "access_token": a, "refresh_token": rt})
}
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req logoutReq
	if !decode(w, r, &req) {
		return
	}
	if req.RefreshToken == "" {
		response.Error(w, r, 400, "refresh_token_required", "refresh token required")
		return
	}
	if err := h.svc.RevokeRefreshToken(r.Context(), req.RefreshToken); err != nil {
		response.Error(w, r, 500, "logout_failed", "could not revoke refresh token")
		return
	}
	if ac, ok := middleware.AuthFromContext(r.Context()); ok {
		h.log(r, ac.UserID, "auth.logout", "user", ac.UserID)
	}
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
func (h *AuthHandler) Sessions(w http.ResponseWriter, r *http.Request) {
	ac, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		response.Error(w, r, 401, "unauthorized", "missing auth context")
		return
	}
	v, err := h.svc.Sessions(r.Context(), ac.UserID)
	if err != nil {
		response.Error(w, r, 500, "sessions_failed", "could not list sessions")
		return
	}
	response.Success(w, r, 200, v)
}
func (h *AuthHandler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	ac, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		response.Error(w, r, 401, "unauthorized", "missing auth context")
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, r, 400, "bad_request", "session id required")
		return
	}
	if err := h.svc.RevokeSession(r.Context(), ac.UserID, id); err != nil {
		response.Error(w, r, 404, "session_not_found", "session not found")
		return
	}
	h.log(r, ac.UserID, "auth.session_revoke", "refresh_token", id)
	response.Success(w, r, 200, map[string]any{"revoked": true})
}
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordReq
	if !decode(w, r, &req) {
		return
	}
	ac, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		response.Error(w, r, 401, "unauthorized", "missing auth context")
		return
	}
	if err := h.svc.ChangePassword(r.Context(), ac.UserID, req.CurrentPassword, req.NewPassword); err != nil {
		response.Error(w, r, 400, "password_change_failed", "could not change password")
		return
	}
	h.log(r, ac.UserID, "auth.password_change", "user", ac.UserID)
	response.Success(w, r, 200, map[string]any{"changed": true, "sessions_revoked": true})
}
func (h *AuthHandler) log(r *http.Request, actor, action, typ, id string) {
	if h.audit == nil {
		return
	}
	_ = h.audit.Log(r.Context(), actor, "", action, typ, id, map[string]any{}, h.clientIP(r), r.UserAgent())
}
func (h *AuthHandler) clientIP(r *http.Request) string {
	return common.ClientIP(r, h.trustProxy)
}
