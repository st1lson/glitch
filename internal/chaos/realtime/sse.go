package realtime

import (
	"bytes"
	"context"
	"math/rand"
	"net/http"
	"sync"

	"github.com/st1lson/glitch/internal/chaos/latency"
	"github.com/st1lson/glitch/internal/config"
)

// SSEInterceptor wraps an http.ResponseWriter to apply chaos to Server-Sent Events.
type SSEInterceptor struct {
	http.ResponseWriter
	config config.RealtimeConfig
	ctx    context.Context

	mu  sync.Mutex
	buf bytes.Buffer

	// Buffered events for out-of-order delivery
	msgQueue [][]byte
}

func NewSSEInterceptor(ctx context.Context, w http.ResponseWriter, cfg config.RealtimeConfig) *SSEInterceptor {
	return &SSEInterceptor{
		ResponseWriter: w,
		config:         cfg,
		ctx:            ctx,
		msgQueue:       make([][]byte, 0),
	}
}

func (s *SSEInterceptor) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buf.Write(p)

	// Parse complete SSE events separated by \n\n delimiters.
	// See: https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events
	for {
		idx := bytes.Index(s.buf.Bytes(), []byte("\n\n"))
		if idx == -1 {
			break
		}

		eventLen := idx + 2
		event := make([]byte, eventLen)
		copy(event, s.buf.Bytes()[:eventLen])
		s.buf.Next(eventLen)

		s.processEvent(event)
	}

	return len(p), nil
}

func (s *SSEInterceptor) processEvent(event []byte) {
	if s.config.DropRate > 0 && rand.Float64()*100 < s.config.DropRate {
		return // Drop event entirely
	}

	if s.config.DisconnectRate > 0 && rand.Float64()*100 < s.config.DisconnectRate {
		panic(http.ErrAbortHandler) // Forcefully drop connection
	}

	if s.config.OutOfOrder {
		maxBuf := s.config.MaxBufferedMessages

		s.msgQueue = append(s.msgQueue, event)

		// If we haven't hit the buffer limit, randomly decide to wait
		if len(s.msgQueue) < maxBuf && rand.Float64() < 0.5 {
			return
		}

		// Pop a random event
		popIdx := rand.Intn(len(s.msgQueue))
		eventToDeliver := s.msgQueue[popIdx]

		// Remove from queue
		s.msgQueue = append(s.msgQueue[:popIdx], s.msgQueue[popIdx+1:]...)
		s.deliverEvent(eventToDeliver)
		return
	}

	s.deliverEvent(event)
}

func (s *SSEInterceptor) deliverEvent(event []byte) {
	if s.config.Latency.Enabled() {
		// Do latency injection synchronously for now to preserve order if OutOfOrder is false,
		// or randomly if it's true.
		// Since this is called from the Write method which is synchronous from ReverseProxy's perspective,
		// adding latency here will stall the reverse proxy copy loop, which correctly applies latency
		// to the stream.
		latency.Inject(s.ctx, s.config.Latency)
	}

	s.ResponseWriter.Write(event)
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *SSEInterceptor) Flush() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Flush any remaining buffered out-of-order messages
	for len(s.msgQueue) > 0 {
		popIdx := rand.Intn(len(s.msgQueue))
		event := s.msgQueue[popIdx]
		s.msgQueue = append(s.msgQueue[:popIdx], s.msgQueue[popIdx+1:]...)
		s.deliverEvent(event)
	}

	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
