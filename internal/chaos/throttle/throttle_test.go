package throttle

import (
	"bytes"
	"net/http/httptest"
	"testing"
	"time"
)

func TestThrottledWriter(t *testing.T) {
	recorder := httptest.NewRecorder()

	bps := 100
	tw := NewWriter(recorder, bps)

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

	expectedMinDuration := 400 * time.Millisecond

	if duration < expectedMinDuration {
		t.Errorf("expected throttle to delay at least %v, took %v", expectedMinDuration, duration)
	}
}
