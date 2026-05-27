package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/response"
)

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		response.Error(w, r, http.StatusBadRequest, "bad_request", "invalid json body")
		return false
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		response.Error(w, r, http.StatusBadRequest, "bad_request", "request body must contain a single json object")
		return false
	}
	return true
}
