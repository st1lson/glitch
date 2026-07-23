package realtime

import (
	"context"
	"net"
	"testing"

	"github.com/st1lson/glitch/internal/config"
)

func TestWSConnInterceptor_ReadWrite(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	cfg := config.RealtimeConfig{}
	
	interceptor := NewWSHijackInterceptor(context.Background(), nil, cfg)
	// We just test the WSConnInterceptor directly since Hijacker is hard to mock without full HTTP server
	conn := &WSConnInterceptor{
		Conn:   server,
		config: interceptor.config,
		ctx:    interceptor.ctx,
	}

	go func() {
		client.Write([]byte("hello"))
		client.Read(make([]byte, 5))
	}()

	buf := make([]byte, 5)
	n, err := conn.Read(buf)
	if err != nil || n != 5 {
		t.Errorf("read failed: %v, %d", err, n)
	}

	n, err = conn.Write(buf)
	if err != nil || n != 5 {
		t.Errorf("write failed: %v, %d", err, n)
	}
}

func TestWSConnInterceptor_Disconnect(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	cfg := config.RealtimeConfig{
		DisconnectRate: 100, // 100% disconnect
	}
	
	conn := &WSConnInterceptor{
		Conn:   server,
		config: cfg,
		ctx:    context.Background(),
	}

	_, err := conn.Read(make([]byte, 5))
	if err != net.ErrClosed {
		t.Errorf("expected net.ErrClosed, got %v", err)
	}
}
