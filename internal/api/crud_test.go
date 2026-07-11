package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/st1lson/glitch/internal/storage"
)

func setupTestRouter(t *testing.T) (chi.Router, *storage.JSONStore) {
	// Create a temporary JSON DB file
	content := `{
		"users": [
			{"id": 1, "name": "Alice"},
			{"id": 2, "name": "Bob"}
		]
	}`
	tmpfile, err := os.CreateTemp("", "testdb-*.json")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(tmpfile.Name()) })

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Create JSON store
	store, err := storage.NewJSONStore(tmpfile.Name(), false)
	if err != nil {
		t.Fatal(err)
	}

	// Setup router
	r := chi.NewRouter()
	RegisterRoutes(r, store)

	return r, store
}

func TestListCollections(t *testing.T) {
	r, _ := setupTestRouter(t)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if resp["users"] != "/users" {
		t.Errorf("expected /users endpoint, got %v", resp["users"])
	}
}

func TestListHandler(t *testing.T) {
	r, _ := setupTestRouter(t)

	req := httptest.NewRequest("GET", "/users", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var items []map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&items); err != nil {
		t.Fatal(err)
	}

	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestGetHandler(t *testing.T) {
	r, _ := setupTestRouter(t)

	req := httptest.NewRequest("GET", "/users/1", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var item map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&item); err != nil {
		t.Fatal(err)
	}

	if item["name"] != "Alice" {
		t.Errorf("expected Alice, got %v", item["name"])
	}
}

func TestCreateHandler(t *testing.T) {
	r, _ := setupTestRouter(t)

	body := []byte(`{"name": "Charlie"}`)
	req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var item map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&item); err != nil {
		t.Fatal(err)
	}

	if item["name"] != "Charlie" {
		t.Errorf("expected Charlie, got %v", item["name"])
	}

	// Verify ID was auto-generated
	if _, ok := item["id"]; !ok {
		t.Error("expected ID to be auto-generated")
	}
}

func TestUpdateHandler(t *testing.T) {
	r, _ := setupTestRouter(t)

	body := []byte(`{"name": "AliceUpdated", "age": 30}`)
	req := httptest.NewRequest("PUT", "/users/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var item map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&item); err != nil {
		t.Fatal(err)
	}

	if item["name"] != "AliceUpdated" {
		t.Errorf("expected AliceUpdated, got %v", item["name"])
	}
}

func TestDeleteHandler(t *testing.T) {
	r, _ := setupTestRouter(t)

	req := httptest.NewRequest("DELETE", "/users/2", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}

	// Verify it's gone
	reqCheck := httptest.NewRequest("GET", "/users/2", nil)
	rrCheck := httptest.NewRecorder()
	r.ServeHTTP(rrCheck, reqCheck)

	if rrCheck.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found, got %v", rrCheck.Code)
	}
}
