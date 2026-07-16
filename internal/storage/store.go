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
	List(collection string) ([]map[string]any, error)
	Get(collection string, id string) (map[string]any, error)
	Create(collection string, item map[string]any) (map[string]any, error)
	Update(collection string, id string, item map[string]any) (map[string]any, error)
	Patch(collection string, id string, fields map[string]any) (map[string]any, error)
	Delete(collection string, id string) error
}
