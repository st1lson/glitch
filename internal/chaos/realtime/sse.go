package realtime

import (
	"bytes"
	"context"
	"net/http"
	"sync"

	"github.com/st1lson/glitch/internal/chaos/latency"
	"github.com/st1lson/glitch/internal/chaos/rng"
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
	if s.config.DropRate > 0 && rng.FromContext(s.ctx).Float64()*100 < s.config.DropRate {
		return
	}

	if s.config.DisconnectRate > 0 && rng.FromContext(s.ctx).Float64()*100 < s.config.DisconnectRate {
		panic(http.ErrAbortHandler)
	}

	if s.config.OutOfOrder {
		maxBuf := s.config.MaxBufferedMessages

		s.msgQueue = append(s.msgQueue, event)

		if len(s.msgQueue) < maxBuf && rng.FromContext(s.ctx).Float64() < 0.5 {
			return
		}

		popIdx := rng.FromContext(s.ctx).IntN(len(s.msgQueue))
		eventToDeliver := s.msgQueue[popIdx]

		s.msgQueue = append(s.msgQueue[:popIdx], s.msgQueue[popIdx+1:]...)
		s.deliverEvent(eventToDeliver)
		return
	}

	s.deliverEvent(event)
}

func (s *SSEInterceptor) deliverEvent(event []byte) {
	if s.config.Latency.Enabled() {

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

	for len(s.msgQueue) > 0 {
		popIdx := rng.FromContext(s.ctx).IntN(len(s.msgQueue))
		event := s.msgQueue[popIdx]
		s.msgQueue = append(s.msgQueue[:popIdx], s.msgQueue[popIdx+1:]...)
		s.deliverEvent(event)
	}

	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
