package handler

import (
	"net/http"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/response"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/media"
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
		Subdir: "avatars",
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
