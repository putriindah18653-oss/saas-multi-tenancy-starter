package common

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP returns RemoteAddr by default. Forwarded headers are trusted only when
// TRUST_PROXY=true is explicitly configured by deployment behind a trusted proxy.
func ClientIP(r *http.Request, trustProxy bool) string {
	if r == nil {
		return ""
	}
	if trustProxy {
		if v := r.Header.Get("X-Forwarded-For"); v != "" {
			for _, part := range strings.Split(v, ",") {
				ip := strings.TrimSpace(part)
				if parsed := net.ParseIP(ip); parsed != nil {
					return parsed.String()
				}
			}
		}
		if v := strings.TrimSpace(r.Header.Get("X-Real-IP")); v != "" {
			if parsed := net.ParseIP(v); parsed != nil {
			return parsed.String()
		}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		if parsed := net.ParseIP(host); parsed != nil {
			return parsed.String()
		}
		return host
	}
	if parsed := net.ParseIP(r.RemoteAddr); parsed != nil {
		return parsed.String()
	}
	return r.RemoteAddr
}
