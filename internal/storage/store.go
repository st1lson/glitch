package storage

import "errors"

var (
	ErrNotFound   = errors.New("resource not found")
	ErrCollection = errors.New("collection not found")
)

// Store defines the interface for data storage backends.
type Store interface {
	// Collections returns the names of all collections.
	Collections() []string
	// List returns all items in a collection.
	List(collection string) ([]map[string]any, error)
	// Get returns a single item by ID.
	Get(collection string, id string) (map[string]any, error)
	// Create adds a new item, auto-generating an ID if not present.
	Create(collection string, item map[string]any) (map[string]any, error)
	// Update replaces an item by ID.
	Update(collection string, id string, item map[string]any) (map[string]any, error)
	// Patch partially updates an item by ID (merge fields).
	Patch(collection string, id string, fields map[string]any) (map[string]any, error)
	// Delete removes an item by ID.
	Delete(collection string, id string) error
}
