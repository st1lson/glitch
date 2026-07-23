package realtime

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/st1lson/glitch/internal/config"
)

func TestSSEInterceptor_Write(t *testing.T) {
	rw := httptest.NewRecorder()
	cfg := config.RealtimeConfig{}
	
	interceptor := NewSSEInterceptor(context.Background(), rw, cfg)

	event1 := "data: hello\n\n"
	event2 := "data: world\n\n"

	// Write in chunks
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
		DropRate: 100, // 100% drop
	}
	
	interceptor := NewSSEInterceptor(context.Background(), rw, cfg)

	interceptor.Write([]byte("data: hello\n\n"))
	interceptor.Flush()

	result := rw.Body.String()
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestSSEInterceptor_OutOfOrder(t *testing.T) {
	rw := httptest.NewRecorder()
	cfg := config.RealtimeConfig{
		OutOfOrder: true,
		MaxBufferedMessages: 2,
	}
	
	interceptor := NewSSEInterceptor(context.Background(), rw, cfg)

	interceptor.Write([]byte("data: 1\n\n"))
	interceptor.Write([]byte("data: 2\n\n"))
	interceptor.Write([]byte("data: 3\n\n"))
	interceptor.Write([]byte("data: 4\n\n"))
	interceptor.Write([]byte("data: 5\n\n"))

	interceptor.Flush()

	result := rw.Body.String()
	
	// Just verify that all messages are present
	if !strings.Contains(result, "data: 1\n\n") || !strings.Contains(result, "data: 5\n\n") {
		t.Errorf("missing messages in out of order delivery: %q", result)
	}
}
