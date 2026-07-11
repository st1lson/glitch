package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/st1lson/glitch/internal/storage"
)

// listHandler returns all items in a collection, applying query-string
// filtering, sorting, and pagination via ApplyQuery.
func listHandler(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		collection := chi.URLParam(r, "collection")
		items, err := store.List(collection)
		if err != nil {
			handleStoreError(w, err)
			return
		}

		// Ensure we never marshal nil as JSON null.
		if items == nil {
			items = []map[string]any{}
		}

		// Apply filtering, sorting, and pagination.
		result, total := ApplyQuery(items, r.URL.Query())

		w.Header().Set("X-Total-Count", strconv.Itoa(total))
		writeJSON(w, http.StatusOK, result)
	}
}

// getHandler returns a single item by ID from the collection.
func getHandler(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		collection := chi.URLParam(r, "collection")
		id := chi.URLParam(r, "id")

		item, err := store.Get(collection, id)
		if err != nil {
			handleStoreError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, item)
	}
}

// createHandler decodes a JSON body and creates a new item in the collection.
func createHandler(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		collection := chi.URLParam(r, "collection")
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err))
			return
		}

		created, err := store.Create(collection, body)
		if err != nil {
			handleStoreError(w, err)
			return
		}

		writeJSON(w, http.StatusCreated, created)
	}
}

// updateHandler decodes a JSON body and replaces the entire item identified by ID.
func updateHandler(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		collection := chi.URLParam(r, "collection")
		id := chi.URLParam(r, "id")

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err))
			return
		}

		updated, err := store.Update(collection, id, body)
		if err != nil {
			handleStoreError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, updated)
	}
}

// patchHandler decodes a JSON body and merges fields into the existing item.
func patchHandler(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		collection := chi.URLParam(r, "collection")
		id := chi.URLParam(r, "id")

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err))
			return
		}

		patched, err := store.Patch(collection, id, body)
		if err != nil {
			handleStoreError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, patched)
	}
}

// deleteHandler removes an item by ID from the collection.
func deleteHandler(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		collection := chi.URLParam(r, "collection")
		id := chi.URLParam(r, "id")

		if err := store.Delete(collection, id); err != nil {
			handleStoreError(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// --- JSON helpers ---

// writeJSON marshals data as JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// At this point headers are already sent; log but can't change the response.
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := map[string]any{
		"error":  message,
		"status": status,
	}
	json.NewEncoder(w).Encode(resp)
}

// handleStoreError maps storage-layer errors to appropriate HTTP responses.
func handleStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, storage.ErrNotFound):
		writeError(w, http.StatusNotFound, "resource not found")
	case errors.Is(err, storage.ErrCollection):
		writeError(w, http.StatusNotFound, "collection not found")
	default:
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("internal error: %v", err))
	}
}
