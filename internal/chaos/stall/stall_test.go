package stall

import (
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

	// Simulate handler setting Content-Length
	payload := []byte("1234567890") // 10 bytes
	sw.Header().Set("Content-Length", "10")
	sw.WriteHeader(http.StatusOK)

	// Threshold is 50% of 10 bytes = 5 bytes.
	// Write 4 bytes
	n, err := sw.Write(payload[:4])
	if n != 4 || err != nil {
		t.Fatalf("first write failed: %d, %v", n, err)
	}

	if rec.Body.Len() != 4 {
		t.Errorf("expected 4 bytes written, got %d", rec.Body.Len())
	}

	// Write the rest, should trigger stall and panic
	_, _ = sw.Write(payload[4:])
}

func TestStallWriter_DropWithoutContentLength(t *testing.T) {
	defer recoverAbort(t, true)

	rec := httptest.NewRecorder()
	sw := NewWriter(rec, config.StallModeDrop, 50)
	sw.WriteHeader(http.StatusOK)

	// Threshold for chunked w/ dropAt 50% is 100KB (102400 bytes).
	chunk := make([]byte, 50*1024)
	sw.Write(chunk) // 50KB written

	sw.Write(chunk) // 100KB written, this should reach the threshold and stall (panic)
}

func TestShouldTrigger(t *testing.T) {
	if ShouldTrigger(config.StallConfig{Rate: 0}) {
		t.Error("Expected false with rate 0")
	}
	if !ShouldTrigger(config.StallConfig{Rate: 100}) {
		t.Error("Expected true with rate 100")
	}
}

func TestStallWriter_DropMode(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := NewWriter(rec, "drop", 1) // drop at 1%
	sw.Header().Set("Content-Length", "100")
	sw.WriteHeader(http.StatusOK) // parses Content-Length -> totalBytes = 100, threshold = 1

	defer func() {
		if r := recover(); r != http.ErrAbortHandler {
			t.Errorf("Expected panic with http.ErrAbortHandler, got %v", r)
		}
	}()
	sw.Write([]byte("test test test")) // len > 1, so it will panic
}
