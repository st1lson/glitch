package throttle

import (
	"net/http"
	"time"
)

// Writer wraps http.ResponseWriter to restrict bandwidth.
type Writer struct {
	http.ResponseWriter
	bps int
}

// NewWriter creates a new Writer.
func NewWriter(w http.ResponseWriter, bps int) http.ResponseWriter {
	return &Writer{
		ResponseWriter: w,
		bps:            bps,
	}
}

// Write handles data streaming and applies pacing.
func (t *Writer) Write(p []byte) (int, error) {
	if t.bps <= 0 {
		return t.ResponseWriter.Write(p)
	}

	// Calculate a chunk size that represents ~100ms of data
	chunkSize := t.bps / 10
	if chunkSize < 1 {
		chunkSize = 1 // Minimum 1 byte per chunk
	}

	totalWritten := 0

	for len(p) > 0 {
		writeSize := chunkSize
		if len(p) < writeSize {
			writeSize = len(p)
		}

		n, err := t.ResponseWriter.Write(p[:writeSize])
		totalWritten += n
		if err != nil {
			return totalWritten, err
		}

		// Flush immediately to the network so the client receives the chunk
		if f, ok := t.ResponseWriter.(http.Flusher); ok {
			f.Flush()
		}

		p = p[writeSize:]

		// Sleep to simulate network delay for this chunk
		if len(p) > 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	return totalWritten, nil
}
