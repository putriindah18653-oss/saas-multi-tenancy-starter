package middleware

import (
	"net/http"
)

// DefaultBodyLimit is the maximum request body size for non-upload endpoints (1 MiB).
// Upload endpoints override this with higher limits via MaxBytesReader.
const DefaultBodyLimit int64 = 1 << 20

// BodyLimit enforces a maximum request body size on all POST/PATCH/PUT requests.
// This prevents memory exhaustion from arbitrarily large JSON payloads.
// Upload endpoints use MaxBytesReader directly with higher limits.
func BodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	if maxBytes <= 0 {
		maxBytes = DefaultBodyLimit
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost || r.Method == http.MethodPatch || r.Method == http.MethodPut {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
