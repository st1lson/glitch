package stall

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/st1lson/glitch/internal/config"
)

func recoverAbort(t *testing.T, expected bool) {
	r := recover()
	if r == nil {
		if expected {
			t.Errorf("expected panic(http.ErrAbortHandler), got nil")
		}
		return
	}
	if r != http.ErrAbortHandler {
		t.Errorf("expected panic(http.ErrAbortHandler), got %v", r)
	} else if !expected {
		t.Errorf("unexpected panic(http.ErrAbortHandler)")
	}
}

func TestStallWriter_DropWithContentLength(t *testing.T) {
	defer recoverAbort(t, true)

	rec := httptest.NewRecorder()
	sw := NewWriter(rec, config.StallModeDrop, 50)

	payload := []byte("1234567890")
	sw.Header().Set("Content-Length", "10")
	sw.WriteHeader(http.StatusOK)

	n, err := sw.Write(payload[:4])
	if n != 4 || err != nil {
		t.Fatalf("first write failed: %d, %v", n, err)
	}

	if rec.Body.Len() != 4 {
		t.Errorf("expected 4 bytes written, got %d", rec.Body.Len())
	}

	_, _ = sw.Write(payload[4:])
}

func TestStallWriter_DropWithoutContentLength(t *testing.T) {
	defer recoverAbort(t, true)

	rec := httptest.NewRecorder()
	sw := NewWriter(rec, config.StallModeDrop, 50)
	sw.WriteHeader(http.StatusOK)

	chunk := make([]byte, 50*1024)
	sw.Write(chunk)

	sw.Write(chunk)
}

func TestShouldTrigger(t *testing.T) {
	if ShouldTrigger(context.Background(), config.StallConfig{Rate: 0}) {
		t.Error("Expected false with rate 0")
	}
	if !ShouldTrigger(context.Background(), config.StallConfig{Rate: 100}) {
		t.Error("Expected true with rate 100")
	}
}

func TestStallWriter_DropMode(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := NewWriter(rec, "drop", 1)
	sw.Header().Set("Content-Length", "100")
	sw.WriteHeader(http.StatusOK)

	defer func() {
		if r := recover(); r != http.ErrAbortHandler {
			t.Errorf("Expected panic with http.ErrAbortHandler, got %v", r)
		}
	}()
	sw.Write([]byte("test test test"))
}
