package httputil

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

// ClientIPKey keys a rate limiter by the proxy-derived client IP set by
// middleware.ClientIPFromXFF (Caddy is the single trusted hop, so the
// rightmost X-Forwarded-For entry is the client) and falls back to
// RemoteAddr for direct (proxyless) connections, e.g. local dev. Keying by
// RemoteAddr alone would put every user behind the proxy into one bucket.
func ClientIPKey(r *http.Request) (string, error) {
	if ip := middleware.GetClientIP(r.Context()); ip != "" {
		return ip, nil
	}
	return httprate.KeyByIP(r)
}
