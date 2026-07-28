package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/st1lson/glitch/internal/chaos"
	"github.com/st1lson/glitch/internal/config"
	"github.com/st1lson/glitch/internal/control"
	"github.com/st1lson/glitch/internal/logging"
)

// NewRouter builds a chi.Router wired with all middleware and API routes.
func NewRouter(state *config.Manager, gate *control.Gatekeeper, apiHandler http.Handler, reporter logging.EventReporter) chi.Router {
	r := chi.NewRouter()

	// Recovery middleware — catch panics and respond with 500.
	r.Use(middleware.Recoverer)

	// CORS — fully permissive for local dev use.
	r.Use(corsMiddleware)

	// Control API — bypasses chaos and pause gate.
	r.Mount("/_glitch", control.NewHandler(state, gate))

	// For the rest of the routes, apply logging, pause gate, and chaos.
	r.Group(func(r chi.Router) {
		r.Use(logging.RequestLogger(state, reporter))

		// Pause gate — blocks non-control requests while paused.
		r.Use(control.PauseMiddleware(gate))

		// Chaos middleware — always mounted so it can be dynamically toggled.
		engine := chaos.NewEngine(state)
		r.Use(engine.Middleware)

		// Mount the specific API handler (JSON, Proxy, or OpenAPI)
		r.Mount("/", apiHandler)
	})

	return r
}

// corsMiddleware is a simple, fully-permissive CORS middleware suitable for
// local development. It allows any origin, common methods, and typical headers.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Glitch-Scenario")
		w.Header().Set("Access-Control-Expose-Headers", "X-Total-Count, X-Glitch-Scenario")
		if scenario := r.Header.Get("X-Glitch-Scenario"); scenario != "" {
			w.Header().Set("X-Glitch-Scenario", scenario)
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
