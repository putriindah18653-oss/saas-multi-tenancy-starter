package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend/internal/common"
)

const RequestIDHeader = common.RequestIDHeader

func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(common.RequestIDContextKey).(string); ok {
		return v
	}
	return ""
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(common.RequestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set(common.RequestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), common.RequestIDContextKey, id)))
	})
}

func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "request-id-unavailable"
	}
	return hex.EncodeToString(b)
}
