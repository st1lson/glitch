package server

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestServer_StartAndShutdown(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Use port 0 to let the OS assign an available random port
	srv := New("127.0.0.1:0", handler)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	// Wait briefly for server to start listening
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := srv.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown() failed: %v", err)
	}

	err = <-errCh
	// Start() returns nil on graceful shutdown
	if err != nil {
		t.Fatalf("Start() returned unexpected error: %v", err)
	}
}
