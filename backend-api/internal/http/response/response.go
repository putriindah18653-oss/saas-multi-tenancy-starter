package response

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/common"
)

type Envelope struct {
	Success   bool       `json:"success"`
	Data      any        `json:"data,omitempty"`
	Error     *ErrorBody `json:"error,omitempty"`
	RequestID string     `json:"request_id,omitempty"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func Success(w http.ResponseWriter, r *http.Request, status int, data any) {
	JSON(w, status, Envelope{Success: true, Data: data, RequestID: RequestID(r)})
}

func Error(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	JSON(w, status, Envelope{Success: false, Error: &ErrorBody{Code: code, Message: message}, RequestID: RequestID(r)})
}

func RequestID(r *http.Request) string {
	if r == nil {
		return ""
	}
	if v, ok := r.Context().Value(common.RequestIDContextKey).(string); ok {
		return v
	}
	return r.Header.Get(common.RequestIDHeader)
}

// LogError logs a structured error message with the request ID for correlation.
func LogError(r *http.Request, msg string, args ...any) {
	attrs := []any{"request_id", RequestID(r)}
	attrs = append(attrs, args...)
	slog.Default().Error(msg, attrs...)
}
