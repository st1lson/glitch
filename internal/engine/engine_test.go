package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func createTempFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "target.file")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNew_ProxyEngine(t *testing.T) {
	// Proxy URL takes precedence
	eng, err := New("", "http://example.com", false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if eng.Name() != "Reverse Proxy" {
		t.Errorf("expected Reverse Proxy, got %s", eng.Name())
	}
	res := eng.Resources()
	if len(res) == 0 || res[0] != "Forwarding to http://example.com" {
		t.Errorf("unexpected resources: %v", res)
	}
	if eng.Handler() == nil {
		t.Errorf("expected non-nil handler")
	}
}

func TestNew_OpenAPIEngine(t *testing.T) {
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`
	path := createTempFile(t, content)

	eng, err := New(path, "", false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if eng.Name() != "OpenAPI Mock Server" {
		t.Errorf("expected OpenAPI Mock Server, got %s", eng.Name())
	}
	res := eng.Resources()
	if len(res) == 0 || res[0] != "Mocking endpoints from "+path {
		t.Errorf("unexpected resources: %v", res)
	}
	if eng.Handler() == nil {
		t.Errorf("expected non-nil handler")
	}
}

func TestNew_JSONEngine(t *testing.T) {
	content := `{"users": []}`
	path := createTempFile(t, content)

	eng, err := New(path, "", false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if eng.Name() != "JSON Database" {
		t.Errorf("expected JSON Database, got %s", eng.Name())
	}
	res := eng.Resources()
	if len(res) != 1 || res[0] != "users" {
		t.Errorf("unexpected resources: %v", res)
	}
	if eng.Handler() == nil {
		t.Errorf("expected non-nil handler")
	}
}

func TestIsOpenAPI(t *testing.T) {
	tests := []struct {
		content []byte
		want    bool
	}{
		{[]byte("openapi: 3.0.0"), true},
		{[]byte("\"openapi\": \"3.0.0\""), true},
		{[]byte("swagger: '2.0'"), true},
		{[]byte("\"swagger\": \"2.0\""), true},
		{[]byte(`{"users": []}`), false},
	}

	for _, tt := range tests {
		if got := isOpenAPI(tt.content); got != tt.want {
			t.Errorf("isOpenAPI(%q) = %v, want %v", string(tt.content), got, tt.want)
		}
	}
}
