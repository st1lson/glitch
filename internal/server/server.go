package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Server wraps an http.Server with sensible defaults and
// provides helpers for starting and gracefully shutting down.
type Server struct {
	httpServer *http.Server
}

// New creates a Server bound to addr with the given handler.
// It sets conservative timeouts suitable for a local dev API server.
func New(addr string, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

// Start begins listening and serving HTTP requests.
// It returns nil when the server is shut down via Shutdown;
// any other error from ListenAndServe is returned as-is.
func (s *Server) Start() error {
	err := s.httpServer.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// Shutdown gracefully shuts down the server without interrupting
// active connections. The provided context controls the deadline.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// StartAndWait starts the server in a goroutine, waits for a SIGINT or SIGTERM,
// and gracefully shuts down.
func (s *Server) StartAndWait() error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		fmt.Printf("\nReceived %v, shutting down...\n", sig)
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	fmt.Println("Server stopped.")
	return nil
}
