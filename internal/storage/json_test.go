package storage

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func createTempDB(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "db.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNewJSONStore(t *testing.T) {
	// Valid DB
	content := `{"users": [{"id": 1, "name": "Alice"}]}`
	path := createTempDB(t, content)

	store, err := NewJSONStore(path, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cols := store.Collections()
	if len(cols) != 1 || cols[0] != "users" {
		t.Errorf("expected ['users'], got %v", cols)
	}

	// Invalid DB
	invalidPath := createTempDB(t, `{"users": [}`)
	_, err = NewJSONStore(invalidPath, true)
	if err == nil {
		t.Error("expected error parsing invalid JSON")
	}

	// Missing file
	_, err = NewJSONStore("/tmp/does/not/exist.json", true)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestJSONStore_CRUD(t *testing.T) {
	content := `{"posts": [{"id": 1, "title": "Hello"}]}`
	path := createTempDB(t, content)
	store, _ := NewJSONStore(path, false)

	// List
	items, err := store.List("posts")
	if err != nil || len(items) != 1 {
		t.Errorf("List failed: %v, items: %v", err, items)
	}

	// Get
	item, err := store.Get("posts", "1")
	if err != nil || item["title"] != "Hello" {
		t.Errorf("Get failed: %v, item: %v", err, item)
	}

	// Get Not Found
	_, err = store.Get("posts", "99")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	// Create
	newItem := map[string]any{"title": "World"}
	created, err := store.Create("posts", newItem)
	if err != nil || created["id"] != 2 {
		t.Errorf("Create failed: %v, item: %v", err, created)
	}

	// Update
	updatedItem := map[string]any{"title": "Updated!"}
	updated, err := store.Update("posts", "1", updatedItem)
	if err != nil || updated["title"] != "Updated!" || updated["id"] != float64(1) {
		t.Errorf("Update failed: %v, item: %v", err, updated)
	}

	// Patch
	patchFields := map[string]any{"author": "Alice"}
	patched, err := store.Patch("posts", "1", patchFields)
	if err != nil || patched["title"] != "Updated!" || patched["author"] != "Alice" {
		t.Errorf("Patch failed: %v, item: %v", err, patched)
	}

	// Delete
	err = store.Delete("posts", "1")
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}
	_, err = store.Get("posts", "1")
	if err != ErrNotFound {
		t.Errorf("expected deleted item to be not found")
	}
}

func TestJSONStore_Concurrency(t *testing.T) {
	content := `{"users": []}`
	path := createTempDB(t, content)
	store, _ := NewJSONStore(path, false)

	var wg sync.WaitGroup
	// Spawn 50 goroutines to concurrently create users
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			store.Create("users", map[string]any{"name": "test"})
		}(i)
	}
	wg.Wait()

	items, _ := store.List("users")
	if len(items) != 50 {
		t.Errorf("expected 50 users, got %d", len(items))
	}
}
