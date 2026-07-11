package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
)

// JSONStore is a file-backed JSON data store that implements the Store interface.
// It keeps the entire database in memory and persists to disk after every mutation.
type JSONStore struct {
	mu       sync.RWMutex
	filePath string
	readOnly bool
	data     map[string][]map[string]any
}

// NewJSONStore loads a JSON file into memory and returns a ready-to-use store.
// The JSON file must contain a top-level object whose keys are collection names
// and whose values are arrays of objects.
func NewJSONStore(filePath string, readOnly bool) (*JSONStore, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading database file: %w", err)
	}

	var data map[string][]map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("parsing database file: %w", err)
	}

	return &JSONStore{
		filePath: filePath,
		readOnly: readOnly,
		data:     data,
	}, nil
}

// Collections returns the names of all collections sorted alphabetically.
func (s *JSONStore) Collections() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.data))
	for name := range s.data {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// List returns all items in a collection.
func (s *JSONStore) List(collection string) ([]map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items, ok := s.data[collection]
	if !ok {
		return nil, ErrCollection
	}

	// Return a copy to prevent external mutation.
	result := make([]map[string]any, len(items))
	for i, item := range items {
		result[i] = copyMap(item)
	}
	return result, nil
}

// Get returns a single item by ID.
func (s *JSONStore) Get(collection string, id string) (map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items, ok := s.data[collection]
	if !ok {
		return nil, ErrCollection
	}

	for _, item := range items {
		if matchID(item["id"], id) {
			return copyMap(item), nil
		}
	}
	return nil, ErrNotFound
}

// Create adds a new item to a collection, auto-generating an ID if one is not present.
// If the collection does not exist, it is created.
func (s *JSONStore) Create(collection string, item map[string]any) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := s.data[collection] // nil is fine; we'll append to it

	newItem := copyMap(item)
	if _, hasID := newItem["id"]; !hasID {
		newItem["id"] = s.nextID(items)
	}

	s.data[collection] = append(items, newItem)

	if err := s.persist(); err != nil {
		return nil, err
	}
	return copyMap(newItem), nil
}

// Update replaces an existing item by ID with the provided item.
func (s *JSONStore) Update(collection string, id string, item map[string]any) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, ok := s.data[collection]
	if !ok {
		return nil, ErrCollection
	}

	for i, existing := range items {
		if matchID(existing["id"], id) {
			updated := copyMap(item)
			updated["id"] = existing["id"] // Preserve the original ID value/type.
			items[i] = updated

			if err := s.persist(); err != nil {
				return nil, err
			}
			return copyMap(updated), nil
		}
	}
	return nil, ErrNotFound
}

// Patch merges the given fields into an existing item identified by ID.
func (s *JSONStore) Patch(collection string, id string, fields map[string]any) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, ok := s.data[collection]
	if !ok {
		return nil, ErrCollection
	}

	for i, existing := range items {
		if matchID(existing["id"], id) {
			for k, v := range fields {
				if k == "id" {
					continue // Don't allow overwriting the ID via patch.
				}
				existing[k] = v
			}
			items[i] = existing

			if err := s.persist(); err != nil {
				return nil, err
			}
			return copyMap(existing), nil
		}
	}
	return nil, ErrNotFound
}

// Delete removes an item by ID from the specified collection.
func (s *JSONStore) Delete(collection string, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, ok := s.data[collection]
	if !ok {
		return ErrCollection
	}

	for i, item := range items {
		if matchID(item["id"], id) {
			s.data[collection] = append(items[:i], items[i+1:]...)

			return s.persist()
		}
	}
	return ErrNotFound
}

// persist writes the current in-memory database to disk as indented JSON.
// Must be called while holding s.mu (write lock).
func (s *JSONStore) persist() error {
	if s.readOnly {
		return nil
	}

	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling database: %w", err)
	}

	if err := os.WriteFile(s.filePath, raw, 0o644); err != nil {
		return fmt.Errorf("writing database file: %w", err)
	}
	return nil
}

// nextID scans the given items for the maximum numeric ID and returns max+1.
// If the collection is empty or has no numeric IDs, it returns 1.
func (s *JSONStore) nextID(items []map[string]any) int {
	maxID := 0
	for _, item := range items {
		switch v := item["id"].(type) {
		case float64:
			if int(v) > maxID {
				maxID = int(v)
			}
		case string:
			if n, err := strconv.Atoi(v); err == nil && n > maxID {
				maxID = n
			}
		}
	}
	return maxID + 1
}

// matchID compares a stored ID value (which may be float64 or string after JSON
// unmarshal) against a string ID (typically from a URL parameter).
// "1" matches both float64(1) and string("1").
func matchID(stored any, target string) bool {
	switch v := stored.(type) {
	case float64:
		// Compare as string: float64(1) -> "1", float64(1.5) -> "1.5".
		return strconv.FormatFloat(v, 'f', -1, 64) == target
	case string:
		return v == target
	default:
		return fmt.Sprint(stored) == target
	}
}

// copyMap creates a shallow copy of a map to prevent external mutation of internal state.
func copyMap(m map[string]any) map[string]any {
	cp := make(map[string]any, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}
