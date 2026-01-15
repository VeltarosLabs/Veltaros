package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

type SecurityConfig struct {
	AllowedOrigins []string        // exact match; "*" not recommended
	APIKey         string          // optional; if set, requires X-API-Key
	RequireKeyFor  map[string]bool // path -> require key
}

func SecurityMiddleware(cfg SecurityConfig, next http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		o = strings.TrimSpace(o)
		if o != "" {
			allowed[o] = struct{}{}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Safe default headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// CORS (only if Origin is present; strict by default)
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Accept,X-API-Key")
			}

			// Preflight
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		// Optional API key enforcement
		if cfg.APIKey != "" && cfg.RequireKeyFor != nil && cfg.RequireKeyFor[r.URL.Path] {
			got := r.Header.Get("X-API-Key")
			if !constantTimeEqualString(got, cfg.APIKey) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"ok":false,"error":"unauthorized"}`))
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func constantTimeEqualString(a, b string) bool {
	ab := []byte(a)
	bb := []byte(b)
	if len(ab) != len(bb) {
		return false
	}
	return subtle.ConstantTimeCompare(ab, bb) == 1
}
