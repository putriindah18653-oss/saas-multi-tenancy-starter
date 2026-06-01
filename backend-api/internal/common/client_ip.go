package common

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP returns RemoteAddr by default. When trustProxy is true, Cloudflare
// headers are considered: CF-Connecting-IP is trusted only when accompanied by
// CF-Ray (injected by Cloudflare on every proxied request). Falls back to
// X-Real-IP and finally RemoteAddr.
func ClientIP(r *http.Request, trustProxy bool) string {
	if r == nil {
		return ""
	}
	if trustProxy {
		// Only trust CF-Connecting-IP when the CF-Ray header is present,
		// confirming the request actually transited through Cloudflare.
		if r.Header.Get("CF-Ray") != "" {
			if v := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); v != "" {
				if parsed := net.ParseIP(v); parsed != nil {
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
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
