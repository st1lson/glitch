package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/st1lson/glitch/internal/chaos"
	"github.com/st1lson/glitch/internal/config"
	"github.com/st1lson/glitch/internal/logging"
)

// NewRouter builds a chi.Router wired with all middleware and API routes.
func NewRouter(cfg config.Config, apiHandler http.Handler) chi.Router {
	r := chi.NewRouter()

	// Recovery middleware — catch panics and respond with 500.
	r.Use(middleware.Recoverer)

	// CORS — fully permissive for local dev use.
	r.Use(corsMiddleware)

	// Request logging.
	r.Use(logging.RequestLogger(cfg.Verbose))

	// Chaos middleware (only if chaos is configured).
	if cfg.HasChaos() {
		engine := chaos.NewEngine(cfg)
		r.Use(engine.Middleware)
	}

	// Mount the specific API handler (JSON, Proxy, or OpenAPI)
	r.Mount("/", apiHandler)

	return r
}

// corsMiddleware is a simple, fully-permissive CORS middleware suitable for
// local development. It allows any origin, common methods, and typical headers.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Expose-Headers", "X-Total-Count")

		// Handle preflight requests.
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
