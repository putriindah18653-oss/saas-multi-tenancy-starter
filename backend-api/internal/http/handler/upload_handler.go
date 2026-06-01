package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/response"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/media"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/middleware"
)

type UploadHandler struct {
	processor media.AVIFProcessor
}

func NewUploadHandler(rootDir string) *UploadHandler {
	return &UploadHandler{processor: media.NewAVIFProcessor(rootDir)}
}

func (h *UploadHandler) AppLogo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, media.DefaultMaxImageBytes)
	if err := r.ParseMultipartForm(media.DefaultMaxImageBytes); err != nil {
		response.Error(w, r, 400, "invalid_upload", "file too large or invalid upload")
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		response.Error(w, r, 400, "image_required", "image file is required")
		return
	}
	defer file.Close()

	ctx, cancel := media.ContextWithConvertTimeout(r.Context())
	defer cancel()
	result, err := h.processor.ConvertUpload(ctx, file, header, media.ConvertOptions{
		Subdir: "app/logos",
		Prefix: "logo",
		Width:  200,
		Height: 200,
	})
	if err != nil {
		response.Error(w, r, 400, "image_convert_failed", "could not convert image to AVIF: "+err.Error())
		return
	}
	response.Success(w, r, 201, result)
}

func (h *UploadHandler) Avatar(w http.ResponseWriter, r *http.Request) {
	ac, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		response.Error(w, r, 401, "unauthorized", "auth required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, media.DefaultMaxImageBytes)
	if err := r.ParseMultipartForm(media.DefaultMaxImageBytes); err != nil {
		response.Error(w, r, 400, "invalid_upload", "file too large or invalid upload")
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		response.Error(w, r, 400, "image_required", "image file is required")
		return
	}
	defer file.Close()

	ctx, cancel := media.ContextWithConvertTimeout(r.Context())
	defer cancel()
	result, err := h.processor.ConvertUpload(ctx, file, header, media.ConvertOptions{
		Subdir: "users/" + ac.UserID + "/avatars",
		Prefix: "avatar",
		Width:  200,
		Height: 200,
	})
	if err != nil {
		response.Error(w, r, 400, "image_convert_failed", "could not convert image to AVIF: "+err.Error())
		return
	}
	response.Success(w, r, 201, result)
}

func (h *UploadHandler) Serve(w http.ResponseWriter, r *http.Request) {
	ac, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		response.Error(w, r, 401, "unauthorized", "auth required")
		return
	}
	path := strings.Trim(chi.URLParam(r, "*"), "/")
	if path == "" || strings.Contains(path, "..") || strings.Contains(path, "\\") {
		response.Error(w, r, 400, "invalid_upload_path", "invalid upload path")
		return
	}
	if !canReadUploadPath(ac.UserID, path) {
		response.Error(w, r, 403, "upload_access_denied", "upload access denied")
		return
	}
	h.processor.Serve(w, r, path)
}

func canReadUploadPath(userID, path string) bool {
	if strings.HasPrefix(path, "app/logos/") {
		return true
	}
	if strings.HasPrefix(path, "users/"+userID+"/avatars/") {
		return true
	}
	// Legacy avatars from before the per-user directory change.
	if strings.HasPrefix(path, "avatars/") {
		return true
	}
	return false
}
