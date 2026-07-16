package throttle

import (
	"bytes"
	"net/http/httptest"
	"testing"
	"time"
)

func TestThrottledWriter(t *testing.T) {
	recorder := httptest.NewRecorder()
	
	// Test: 100 bytes per second limit
	bps := 100
	tw := newThrottledWriter(recorder, bps)

	// Payload is exactly 50 bytes.
	// We expect this to be written in chunks.
	// Since 50 bytes < 100 bps, the loop calculates chunkSize = bps/10 = 10 bytes.
	// It will write 5 chunks of 10 bytes, sleeping ~100ms each time.
	// So 50 bytes should take roughly 500ms to send.
	payload := make([]byte, 50)
	for i := range payload {
		payload[i] = 'A'
	}

	start := time.Now()
	n, err := tw.Write(payload)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if n != len(payload) {
		t.Errorf("expected to write %d bytes, wrote %d", len(payload), n)
	}

	if !bytes.Equal(recorder.Body.Bytes(), payload) {
		t.Errorf("recorded body doesn't match payload")
	}

	// 5 iterations of 100ms = ~500ms
	// Allow some wiggle room for execution time.
	expectedMinDuration := 400 * time.Millisecond
	
	if duration < expectedMinDuration {
		t.Errorf("expected throttle to delay at least %v, took %v", expectedMinDuration, duration)
	}
}
