package handler

import (
	"encoding/json"
	"net/http"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/response"
)

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		response.Error(w, r, http.StatusBadRequest, "bad_request", "invalid json body")
		return false
	}
	return true
}
