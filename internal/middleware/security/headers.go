package security

import (
	"net/http"
	"strings"

	"github.com/manto/manto-web/internal/config"
)

func SecurityHeaders(cfg *config.Config) func(http.Handler) http.Handler {
	allowed := strings.Join(cfg.Security.AllowedAPIEndpoints, " ")
	csp := "default-src 'self'; " +
		"connect-src 'self' " + allowed + "; " +
		"style-src 'self' 'unsafe-inline'; " +
		"script-src 'self'; " +
		"img-src 'self' data:; " +
		"object-src 'none'; base-uri 'self'"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("Permissions-Policy", "geolocation=()")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Cross-Origin-Resource-Policy", "same-site")
			w.Header().Set("Content-Security-Policy", csp)

			if cfg.Security.EnableHSTS {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			}

			next.ServeHTTP(w, r)
		})
	}
}
