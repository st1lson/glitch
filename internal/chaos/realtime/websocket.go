package realtime

import (
	"bufio"
	"context"
	"math/rand"
	"net"
	"net/http"

	"github.com/st1lson/glitch/internal/chaos/latency"
	"github.com/st1lson/glitch/internal/config"
)

// WSHijackInterceptor wraps an http.ResponseWriter that implements http.Hijacker.
// It intercepts the Hijack call to return a decorated net.Conn for WebSocket chaos.
type WSHijackInterceptor struct {
	http.ResponseWriter
	config config.RealtimeConfig
	ctx    context.Context
}

func NewWSHijackInterceptor(ctx context.Context, w http.ResponseWriter, cfg config.RealtimeConfig) *WSHijackInterceptor {
	return &WSHijackInterceptor{
		ResponseWriter: w,
		config:         cfg,
		ctx:            ctx,
	}
}

// Hijack intercepts the hijacking to wrap the net.Conn.
func (w *WSHijackInterceptor) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}

	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, nil, err
	}
	
	// Flush any pending data
	rw.Flush()

	wrappedConn := &WSConnInterceptor{
		Conn:   conn,
		config: w.config,
		ctx:    w.ctx,
	}

	// We create a new bufio.ReadWriter using the wrapped connection
	// so that subsequent reads/writes go through our chaos logic.
	wrappedRW := bufio.NewReadWriter(bufio.NewReader(wrappedConn), bufio.NewWriter(wrappedConn))

	return wrappedConn, wrappedRW, nil
}

// WSConnInterceptor wraps a net.Conn to apply Latency and Disconnects.
// TODO: Implement out-of-order and dropped messages using a proper WebSocket frame parser package.
type WSConnInterceptor struct {
	net.Conn
	config config.RealtimeConfig
	ctx    context.Context
}

func (w *WSConnInterceptor) Read(b []byte) (n int, err error) {
	if w.config.DisconnectRate > 0 && rand.Float64()*100 < w.config.DisconnectRate {
		w.Conn.Close()
		return 0, net.ErrClosed
	}

	if w.config.Latency.Enabled() {
		latency.Inject(w.ctx, w.config.Latency)
	}

	return w.Conn.Read(b)
}

func (w *WSConnInterceptor) Write(b []byte) (n int, err error) {
	if w.config.DisconnectRate > 0 && rand.Float64()*100 < w.config.DisconnectRate {
		w.Conn.Close()
		return 0, net.ErrClosed
	}

	if w.config.Latency.Enabled() {
		latency.Inject(w.ctx, w.config.Latency)
	}

	return w.Conn.Write(b)
}
