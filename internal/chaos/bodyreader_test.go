package chaos

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPeekJSONBody(t *testing.T) {
	jsonPayload := `{"user": {"name": "Alice", "role": "admin", "age": 30, "active": true}, "tags": ["lead", "eng"]}`
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(jsonPayload))

	data, updatedReq := PeekJSONBody(req)
	if data == nil {
		t.Fatal("expected parsed JSON data")
	}

	// Verify downstream reading still works!
	bodyBytes, err := io.ReadAll(updatedReq.Body)
	if err != nil {
		t.Fatalf("failed to read body after peek: %v", err)
	}
	if string(bodyBytes) != jsonPayload {
		t.Errorf("body content mismatch: got %q, want %q", string(bodyBytes), jsonPayload)
	}

	// Test caching in context
	data2, _ := PeekJSONBody(updatedReq)
	if data2 == nil || data2["user"] == nil {
		t.Error("expected cached data in context")
	}
}

func TestPeekJSONBody_EmptyOrInvalid(t *testing.T) {
	reqEmpty := httptest.NewRequest(http.MethodGet, "/test", nil)
	data, _ := PeekJSONBody(reqEmpty)
	if data != nil {
		t.Error("expected nil data for empty body")
	}

	reqInvalid := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString("not json"))
	dataInv, _ := PeekJSONBody(reqInvalid)
	if dataInv != nil {
		t.Error("expected nil data for invalid json")
	}
}

func TestGetNestedValue(t *testing.T) {
	data := map[string]any{
		"a": map[string]any{
			"b": map[string]any{
				"c": "deep-value",
			},
		},
		"top": 123,
	}

	val, ok := GetNestedValue(data, "a.b.c")
	if !ok || val != "deep-value" {
		t.Errorf("expected deep-value, got %v (ok=%v)", val, ok)
	}

	valTop, okTop := GetNestedValue(data, "top")
	if !okTop || valTop != 123 {
		t.Errorf("expected 123, got %v", valTop)
	}

	_, okNone := GetNestedValue(data, "a.b.missing")
	if okNone {
		t.Error("expected false for non-existent field")
	}

	_, okInvalid := GetNestedValue(data, "top.child")
	if okInvalid {
		t.Error("expected false when traversing non-map")
	}
}

func TestEvaluateBodyCondition(t *testing.T) {
	tests := []struct {
		name     string
		actual   any
		exists   bool
		op       string
		expected string
		want     bool
	}{
		{"eq string true", "admin", true, "eq", "admin", true},
		{"eq string case insensitive", "Admin", true, "eq", "admin", true},
		{"eq string false", "user", true, "eq", "admin", false},
		{"eq number", float64(42), true, "eq", "42", true},
		{"eq bool", true, true, "eq", "true", true},
		{"eq missing", nil, false, "eq", "val", false},

		{"neq true", "user", true, "neq", "admin", true},
		{"neq false", "admin", true, "neq", "admin", false},
		{"neq missing", nil, false, "neq", "admin", true},

		{"exists true", "anything", true, "exists", "", true},
		{"exists false", nil, false, "exists", "", false},
		{"exists false-check true", nil, false, "exists", "false", true},
		{"exists false-check false", "val", true, "exists", "false", false},

		{"contains string", "hello world", true, "contains", "world", true},
		{"contains slice", []any{"read", "write"}, true, "contains", "write", true},
		{"contains slice missing", []any{"read", "write"}, true, "contains", "delete", false},

		{"prefix true", "Bearer 12345", true, "prefix", "Bearer", true},
		{"prefix false", "Token 12345", true, "prefix", "Bearer", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateBodyCondition(tt.actual, tt.exists, tt.op, tt.expected)
			if got != tt.want {
				t.Errorf("EvaluateBodyCondition(%v, %v, %q, %q) = %v, want %v",
					tt.actual, tt.exists, tt.op, tt.expected, got, tt.want)
			}
		})
	}
}
