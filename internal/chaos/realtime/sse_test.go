package realtime

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/st1lson/glitch/internal/chaos/rng"
	"github.com/st1lson/glitch/internal/config"
)

func TestSSEInterceptor_Write(t *testing.T) {
	rw := httptest.NewRecorder()
	cfg := config.RealtimeConfig{}

	interceptor := NewSSEInterceptor(context.Background(), rw, cfg)

	event1 := "data: hello\n\n"
	event2 := "data: world\n\n"

	interceptor.Write([]byte("data: he"))
	interceptor.Write([]byte("llo\n\n"))
	interceptor.Write([]byte("data: world\n\n"))

	interceptor.Flush()

	result := rw.Body.String()
	expected := event1 + event2

	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSSEInterceptor_Drop(t *testing.T) {
	rw := httptest.NewRecorder()
	cfg := config.RealtimeConfig{
		DropRate: 100,
	}

	interceptor := NewSSEInterceptor(context.Background(), rw, cfg)

	interceptor.Write([]byte("data: hello\n\n"))
	interceptor.Flush()

	result := rw.Body.String()
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

// seededContext makes the shuffling under test reproducible.
func seededContext(seed int64) context.Context {
	return rng.WithRNG(context.Background(), rng.New(seed))
}

func writeEvents(interceptor *SSEInterceptor, count int) []string {
	events := make([]string, 0, count)
	for i := 1; i <= count; i++ {
		event := fmt.Sprintf("data: %d\n\n", i)
		events = append(events, event)
		interceptor.Write([]byte(event))

		// A real SSE producer flushes after every event it writes.
		interceptor.Flush()
	}
	return events
}

// deliverOutOfOrder runs one stream through the interceptor under a fixed seed.
func deliverOutOfOrder(cfg config.RealtimeConfig, seed int64, count int) (events []string, result string) {
	rw := httptest.NewRecorder()
	interceptor := NewSSEInterceptor(seededContext(seed), rw, cfg)
	events = writeEvents(interceptor, count)
	interceptor.Drain()
	return events, rw.Body.String()
}

// Sweeps fixed seeds so the test stays deterministic without depending on one
// seed that happens to shuffle.
func TestSSEInterceptor_OutOfOrder(t *testing.T) {
	cfg := config.RealtimeConfig{
		OutOfOrder:          true,
		MaxBufferedMessages: 3,
	}

	reordered := false
	for seed := int64(1); seed <= 20; seed++ {
		events, result := deliverOutOfOrder(cfg, seed, 8)

		for _, event := range events {
			if !strings.Contains(result, event) {
				t.Errorf("seed %d: missing %q in out of order delivery: %q", seed, event, result)
			}
		}

		if result != strings.Join(events, "") {
			reordered = true
		}
	}

	if !reordered {
		t.Error("out of order delivery never changed the order of a stream")
	}
}

// Regression guard. Flush used to drain the whole queue, and because an SSE
// producer flushes after every event, that handed each held event straight back
// in arrival order. Out-of-order delivery then did nothing at all.
func TestSSEInterceptor_FlushDoesNotDrainQueue(t *testing.T) {
	cfg := config.RealtimeConfig{
		OutOfOrder:          true,
		MaxBufferedMessages: 100,
	}

	heldSomething := false
	for seed := int64(1); seed <= 20; seed++ {
		rw := httptest.NewRecorder()
		interceptor := NewSSEInterceptor(seededContext(seed), rw, cfg)
		events := writeEvents(interceptor, 8)

		if rw.Body.String() != strings.Join(events, "") {
			heldSomething = true
		}

		interceptor.Drain()

		for _, event := range events {
			if !strings.Contains(rw.Body.String(), event) {
				t.Errorf("seed %d: draining did not deliver %q: %q", seed, event, rw.Body.String())
			}
		}
	}

	if !heldSomething {
		t.Error("flushing delivered every event in order, leaving nothing to reorder")
	}
}

// An unset MaxBufferedMessages must fall back to a default rather than silently
// disabling the feature.
func TestSSEInterceptor_OutOfOrderWithoutBufferLimit(t *testing.T) {
	cfg := config.RealtimeConfig{OutOfOrder: true}

	reordered := false
	for seed := int64(1); seed <= 20; seed++ {
		events, result := deliverOutOfOrder(cfg, seed, 8)

		for _, event := range events {
			if !strings.Contains(result, event) {
				t.Errorf("seed %d: missing %q: %q", seed, event, result)
			}
		}

		if result != strings.Join(events, "") {
			reordered = true
		}
	}

	if !reordered {
		t.Error("an unset MaxBufferedMessages disabled out of order delivery")
	}
}
