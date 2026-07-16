package corruption

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/st1lson/glitch/internal/config"
)

func TestShouldTrigger(t *testing.T) {
	// Rate 0 should never corrupt
	if ShouldTrigger(config.CorruptionConfig{Rate: 0}) {
		t.Error("Expected false with rate 0")
	}

	// Rate 100 should always corrupt
	if !ShouldTrigger(config.CorruptionConfig{Rate: 100}) {
		t.Error("Expected true with rate 100")
	}
}

func TestFieldDropper(t *testing.T) {
	m := &FieldDropper{}

	// Object
	obj := map[string]any{"a": 1, "b": 2}
	res := m.Mutate(obj).(map[string]any)
	if len(res) != 1 {
		t.Errorf("Expected 1 key, got %d", len(res))
	}

	// Array
	arr := []any{1, 2, 3}
	resArr := m.Mutate(arr).([]any)
	if len(resArr) != 2 {
		t.Errorf("Expected length 2, got %d", len(resArr))
	}
}

func TestTypeSwapper(t *testing.T) {
	m := &TypeSwapper{}

	// Object
	obj := map[string]any{"a": "string"}
	res := m.Mutate(obj).(map[string]any)
	if _, ok := res["a"].(int); !ok {
		t.Error("Expected string to be swapped to int")
	}

	// Array
	arr := []any{true}
	resArr := m.Mutate(arr).([]any)
	if _, ok := resArr[0].(int); !ok {
		t.Error("Expected bool to be swapped to int")
	}
}

func TestNullInjector(t *testing.T) {
	m := &NullInjector{}

	// Object
	obj := map[string]any{"a": "string"}
	res := m.Mutate(obj).(map[string]any)
	if res["a"] != nil {
		t.Error("Expected value to be null")
	}

	// Array
	arr := []any{1}
	resArr := m.Mutate(arr).([]any)
	if resArr[0] != nil {
		t.Error("Expected value to be null")
	}
}

func TestSyntaxBreaker(t *testing.T) {
	m := &SyntaxBreaker{}
	
	validJSON := []byte(`{"a": 1}`)
	res := m.Mutate(validJSON).([]byte)

	var dump any
	if err := json.Unmarshal(res, &dump); err == nil {
		t.Error("Expected syntax breaker to invalidate JSON")
	}
}

func TestCorruptPayload(t *testing.T) {
	validJSON := []byte(`{"nested": {"a": 1}}`)
	cfg := config.CorruptionConfig{
		Strategies: []string{"drop_field"},
	}

	res, name := CorruptPayload(validJSON, cfg)
	if bytes.Equal(validJSON, res) {
		t.Error("Expected payload to be mutated")
	}
	if name != "drop_field" {
		t.Errorf("Expected name 'drop_field', got %s", name)
	}

	// Test multi
	cfgMulti := config.CorruptionConfig{
		Multi: true,
	}
	resMulti, nameMulti := CorruptPayload(validJSON, cfgMulti)
	if bytes.Equal(validJSON, resMulti) {
		t.Error("Expected payload to be mutated in multi mode")
	}
	if nameMulti == "" {
		t.Error("Expected mutator names")
	}
}

func TestCorruptionWriter_JSON(t *testing.T) {
	cfg := config.CorruptionConfig{
		Strategies: []string{"inject_null"},
	}

	rec := httptest.NewRecorder()
	cw := NewWriter(rec, cfg)

	cw.Header().Set("Content-Type", "application/json")
	cw.WriteHeader(http.StatusOK)
	cw.Write([]byte(`{"a": 1}`))
	cw.Flush()

	res := rec.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", res.StatusCode)
	}

	body := rec.Body.Bytes()
	if bytes.Equal(body, []byte(`{"a": 1}`)) {
		t.Error("Expected body to be mutated")
	}

	cl := res.Header.Get("Content-Length")
	if cl == "" || cl == "8" {
		t.Errorf("Expected Content-Length to be updated, got %s", cl)
	}
}

func TestCorruptionWriter_NonJSON(t *testing.T) {
	cfg := config.CorruptionConfig{
		Strategies: []string{"inject_null"},
	}

	rec := httptest.NewRecorder()
	cw := NewWriter(rec, cfg)

	cw.Header().Set("Content-Type", "text/plain")
	cw.WriteHeader(http.StatusOK)
	cw.Write([]byte(`{"a": 1}`))
	cw.Flush()

	res := rec.Result()
	_ = res
	body := rec.Body.Bytes()

	if !bytes.Equal(body, []byte(`{"a": 1}`)) {
		t.Error("Expected non-JSON body to be passed through untouched")
	}
}
