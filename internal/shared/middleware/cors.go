package middleware

import (
	"net/http"

	"github.com/go-chi/cors"
)

// CORSOptions builds chi/cors options with safe credential handling.
// Uses AllowOriginFunc to reflect the exact requesting origin instead of *.
func CORSOptions(origins []string) cors.Options {
	opts := cors.Options{
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-Organization-ID", "X-App-ID"},
		ExposedHeaders: []string{"X-Request-ID"},
		MaxAge:         300,
	}

	// Build origin set for fast lookup
	originSet := make(map[string]struct{}, len(origins))
	wildcard := false
	for _, origin := range origins {
		if origin == "*" {
			wildcard = true
			break
		}
		originSet[origin] = struct{}{}
	}

	switch {
	case len(origins) == 0:
		// No origins configured: deny all cross-origin
		opts.AllowCredentials = false
		opts.AllowOriginFunc = func(r *http.Request, origin string) bool {
			return false
		}

	case wildcard:
		// Wildcard configured: accept any origin, reflect it back
		opts.AllowCredentials = true
		opts.AllowOriginFunc = func(r *http.Request, origin string) bool {
			return origin != ""
		}

	default:
		// Specific origins: match against set, reflect matching origin
		opts.AllowCredentials = true
		opts.AllowOriginFunc = func(r *http.Request, origin string) bool {
			_, ok := originSet[origin]
			return ok
		}
	}

	return opts
}
