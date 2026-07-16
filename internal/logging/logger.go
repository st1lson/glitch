package logging

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/st1lson/glitch/internal/chaos"
	"github.com/st1lson/glitch/internal/config"
)

// LogEvent contains all information about a processed HTTP request.
type LogEvent struct {
	Timestamp      time.Time
	Method         string
	Path           string
	StatusCode     int
	Duration       time.Duration
	ChaosLatency   time.Duration
	ChaosFailure   int
	ChaosCorrupted bool
	Formatted      string
}

// EventReporter is an interface for receiving log events asynchronously.
type EventReporter interface {
	Report(event LogEvent)
}

// RequestLogger returns a chi-compatible middleware that logs every HTTP request.
// If an EventReporter is provided, events are broadcasted to it.
// Otherwise, events are printed directly to stdout.
func RequestLogger(state *config.State, reporter EventReporter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			verbose := state.Get().Verbose

			// In verbose mode, capture the request body so we can log it.
			var bodyBytes []byte
			if verbose {
				if r.Body != nil {
					bodyBytes, _ = io.ReadAll(r.Body)
					r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				}
			}

			// Wrap the response writer to capture the status code and bytes written.
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			// Build the log line.
			method := colorMethod(r.Method)
			status := colorStatus(rw.statusCode)
			path := r.URL.RequestURI()

			line := fmt.Sprintf("%s  %s  %s  %s", method, path, status, formatDuration(duration))

			event := LogEvent{
				Timestamp:  start,
				Method:     r.Method,
				Path:       r.URL.RequestURI(),
				StatusCode: rw.statusCode,
				Duration:   duration,
			}

			// Append chaos annotations if present.
			if info := chaos.GetChaosInfo(r); info != nil {
				event.ChaosLatency = info.LatencyAdded
				event.ChaosFailure = info.FailureCode
				event.ChaosCorrupted = info.Corrupted

				annotations := buildChaosAnnotations(info)
				if annotations != "" {
					line += "  " + annotations
				}
			}

			event.Formatted = line

			if reporter != nil {
				reporter.Report(event)
			} else {
				fmt.Println(line)
			}

			// Verbose: print headers and body.
			if verbose && reporter == nil {
				printVerbose(r, bodyBytes)
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code
// and the number of bytes written to the response body.
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	wroteHeader  bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.wroteHeader = true
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// Unwrap implements the http.ResponseController interface for compatibility
// with middleware that wraps response writers (e.g., chi's middleware stack).
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// --- Color helpers ---

// colorMethod returns the HTTP method string colored by convention.
func colorMethod(method string) string {
	padded := fmt.Sprintf("%-7s", method)
	switch method {
	case http.MethodGet:
		return color.GreenString(padded)
	case http.MethodPost:
		return color.CyanString(padded)
	case http.MethodPut:
		return color.YellowString(padded)
	case http.MethodPatch:
		return color.YellowString(padded)
	case http.MethodDelete:
		return color.RedString(padded)
	default:
		return padded
	}
}

// colorStatus returns the status code string colored by class.
func colorStatus(code int) string {
	s := fmt.Sprintf("%d", code)
	switch {
	case code >= 200 && code < 300:
		return color.GreenString(s)
	case code >= 400 && code < 500:
		return color.YellowString(s)
	case code >= 500:
		return color.RedString(s)
	default:
		return s
	}
}

// formatDuration returns a human-friendly duration string.
func formatDuration(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return fmt.Sprintf("%dµs", d.Microseconds())
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

// buildChaosAnnotations creates the annotation string for chaos injection info.
func buildChaosAnnotations(info *chaos.ChaosInfo) string {
	var parts []string

	if info.LatencyAdded > 0 {
		parts = append(parts, fmt.Sprintf("⚡ +%dms latency", info.LatencyAdded.Milliseconds()))
	}

	if info.FailureCode > 0 {
		parts = append(parts, fmt.Sprintf("💥 injected %d", info.FailureCode))
	}

	if info.Corrupted {
		parts = append(parts, "🧬 corrupted")
	}

	return strings.Join(parts, "  ")
}

// printVerbose prints request headers and body for verbose logging.
func printVerbose(r *http.Request, body []byte) {
	gray := color.New(color.FgHiBlack)

	gray.Println("  Headers:")
	for name, values := range r.Header {
		for _, v := range values {
			gray.Printf("    %s: %s\n", name, v)
		}
	}

	if len(body) > 0 {
		gray.Println("  Body:")
		gray.Printf("    %s\n", string(body))
	}
}
