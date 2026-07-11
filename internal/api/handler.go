package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/st1lson/glitch/internal/storage"
)

// RegisterRoutes mounts generic REST-style CRUD routes.
// It uses URL parameters so any collection name is supported dynamically.
func RegisterRoutes(r chi.Router, store storage.Store) {
	// Root endpoint: list all available collections with their paths.
	r.Get("/", rootHandler(store))

	// Generic collection routes
	r.Route("/{collection}", func(cr chi.Router) {
		cr.Get("/", listHandler(store))
		cr.Post("/", createHandler(store))
		
		// Item specific routes
		cr.Get("/{id}", getHandler(store))
		cr.Put("/{id}", updateHandler(store))
		cr.Patch("/{id}", patchHandler(store))
		cr.Delete("/{id}", deleteHandler(store))
	})
}

// rootHandler returns a JSON object mapping each collection name to its endpoint path.
func rootHandler(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		collections := store.Collections()
		endpoints := make(map[string]string, len(collections))
		for _, col := range collections {
			endpoints[col] = fmt.Sprintf("/%s", col)
		}
		writeJSON(w, http.StatusOK, endpoints)
	}
}
