package stall

import (
	"math/rand/v2"
	"net/http"
	"strconv"

	"github.com/st1lson/glitch/internal/config"
)

// ShouldTrigger determines if the request should be stalled based on the configured rate.
func ShouldTrigger(cfg config.StallConfig) bool {
	if cfg.Rate <= 0 {
		return false
	}
	return rand.Float64() < (cfg.Rate / 100.0)
}

// Writer wraps an http.ResponseWriter to simulate a stall (hang or drop)
// during the response body stream.
type Writer struct {
	http.ResponseWriter
	mode         config.StallMode
	dropAtRatio  float64
	totalBytes   int64
	writtenBytes int64
	stalled      bool
}

// NewWriter creates a new Writer.
func NewWriter(w http.ResponseWriter, mode config.StallMode, dropAt float64) http.ResponseWriter {
	if dropAt <= 0 {
		dropAt = 50 // default to 50%
	}
	if mode == "" {
		mode = config.StallModeDrop // default mode
	}
	return &Writer{
		ResponseWriter: w,
		mode:           mode,
		dropAtRatio:    float64(dropAt) / 100.0,
	}
}

// WriteHeader overrides the default WriteHeader.
func (s *Writer) WriteHeader(statusCode int) {
	cl := s.Header().Get("Content-Length")
	if cl != "" {
		if parsed, err := strconv.ParseInt(cl, 10, 64); err == nil {
			s.totalBytes = parsed
		}
	}
	s.ResponseWriter.WriteHeader(statusCode)
}

// Write writes data and simulates stall conditions when crossing the drop threshold.
func (s *Writer) Write(p []byte) (int, error) {
	if s.stalled {
		if s.mode == config.StallModeHang {
			select {} // block indefinitely
		}
		panic(http.ErrAbortHandler)
	}

	writeSize := len(p)
	var stallAfterWrite bool

	// Determine threshold
	var threshold int64
	if s.totalBytes > 0 {
		threshold = int64(float64(s.totalBytes) * s.dropAtRatio)
	} else {
		// If size is unknown (e.g., chunked), we drop after an arbitrary number of bytes.
		// For a default dropAtRatio of 0.5 (50%), we simulate a 200KB payload
		// stopping at 100KB.
		threshold = int64(200 * 1024 * s.dropAtRatio)
	}

	if threshold <= 0 {
		threshold = 1 // Drop after at least 1 byte if configured 0%
	}

	if s.writtenBytes+int64(writeSize) >= threshold {
		// We should stall during this write.
		allowed := threshold - s.writtenBytes
		if allowed < 0 {
			allowed = 0
		}
		writeSize = int(allowed)
		stallAfterWrite = true
	}

	n, err := s.ResponseWriter.Write(p[:writeSize])
	s.writtenBytes += int64(n)

	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}

	if err != nil {
		return n, err
	}

	if stallAfterWrite {
		s.stalled = true
		if s.mode == config.StallModeHang {
			select {} // block indefinitely
		}
		panic(http.ErrAbortHandler)
	}

	return n, nil
}
