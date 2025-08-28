package security

import (
	"net/http"
	"strings"

	"github.com/manto/manto-web/internal/config"
)

func SecurityHeaders(cfg *config.Config) func(http.Handler) http.Handler {
	allowedEndpoints := strings.Join(cfg.Security.AllowedAPIEndpoints, " ")
	cspHeader := "default-src 'self'; " +
		"connect-src 'self' " + allowedEndpoints + "; " +
		"style-src 'self' 'unsafe-inline'; " +
		"script-src 'self'; " +
		"img-src 'self'; " +
		"object-src 'none'; " +
		"base-uri 'self'"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("Permissions-Policy", "geolocation=()")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
			w.Header().Set("Cross-Origin-Resource-Policy", "same-site")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
			w.Header().Set("Content-Security-Policy", cspHeader)
			next.ServeHTTP(w, r)
		})
	}
}
