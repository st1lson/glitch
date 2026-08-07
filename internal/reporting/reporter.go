package reporting

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/st1lson/glitch/internal/config"
	"github.com/st1lson/glitch/internal/constants"
	"github.com/st1lson/glitch/internal/logging"
)

const maxRequestEvents = 1000

// RequestEvent is the JSON representation of an individual request.
type RequestEvent struct {
	Timestamp        time.Time `json:"timestamp"`
	Method           string    `json:"method"`
	Path             string    `json:"path"`
	Status           int       `json:"status"`
	DurationMs       int64     `json:"duration_ms"`
	ChaosLatencyMs   int64     `json:"chaos_latency_ms,omitempty"`
	ChaosFailureCode int       `json:"chaos_failure_code,omitempty"`
	ChaosCorrupted   bool      `json:"chaos_corrupted,omitempty"`
}

// Metrics tracks aggregate sums/counts.
type Metrics struct {
	Requests            int64 `json:"requests"`
	Failures            int64 `json:"failures"`
	Stalls              int64 `json:"stalls"`
	TotalLatencyAddedMs int64 `json:"total_latency_added_ms"`
	CorruptedPayloads   int64 `json:"corrupted_payloads"`
	TotalBytesWritten   int64 `json:"total_bytes_written"`
	TotalDurationMs     int64 `json:"total_duration_ms"`
}

// ScenarioReport represents the full report for a scenario.
type ScenarioReport struct {
	mu              sync.RWMutex   `json:"-"`
	head            int            `json:"-"`
	Scenario        string         `json:"scenario"`
	Seed            *int64         `json:"seed"`
	EffectiveConfig config.Config  `json:"effective_config"`
	Metrics         Metrics        `json:"metrics"`
	RequestEvents   []RequestEvent `json:"request_events"`
	Status          string         `json:"status"`
}

// Clone creates a thread-safe, chronological copy of the report with dynamically hydrated config.
func (sr *ScenarioReport) Clone(cfg config.Config) ScenarioReport {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	res := *sr
	res.EffectiveConfig = cfg

	res.RequestEvents = make([]RequestEvent, 0, len(sr.RequestEvents))
	if len(sr.RequestEvents) < maxRequestEvents {
		res.RequestEvents = append(res.RequestEvents, sr.RequestEvents...)
	} else {
		idx := sr.head % maxRequestEvents
		res.RequestEvents = append(res.RequestEvents, sr.RequestEvents[idx:]...)
		res.RequestEvents = append(res.RequestEvents, sr.RequestEvents[:idx]...)
	}
	return res
}

// ReportManager aggregates metrics and RequestEvents per scenario.
type ReportManager struct {
	mu        sync.RWMutex
	state     *config.Manager
	scenarios map[string]*ScenarioReport
}

// NewReportManager creates a new metrics collector.
func NewReportManager(state *config.Manager) *ReportManager {
	return &ReportManager{
		state:     state,
		scenarios: make(map[string]*ScenarioReport),
	}
}

func (r *ReportManager) getOrCreate(scenarioName string) *ScenarioReport {
	r.mu.Lock()
	defer r.mu.Unlock()

	if sr, ok := r.scenarios[scenarioName]; ok {
		return sr
	}

	cfg := r.state.Resolve(scenarioName)
	sr := &ScenarioReport{
		Scenario:      scenarioName,
		Seed:          cfg.Seed,
		Status:        constants.StatusActive,
		RequestEvents: make([]RequestEvent, 0, maxRequestEvents),
	}
	r.scenarios[scenarioName] = sr
	return sr
}

// Report handles incoming log events from the request logger.
func (r *ReportManager) Report(event logging.LogEvent) {
	scenario := event.Scenario

	if scenario == "" || !r.state.Has(scenario) {
		scenario = constants.DefaultScenario
	}

	r.mu.RLock()
	sr, ok := r.scenarios[scenario]
	r.mu.RUnlock()

	if !ok {
		sr = r.getOrCreate(scenario)
	}

	sr.mu.Lock()
	defer sr.mu.Unlock()

	sr.Metrics.Requests++
	sr.Metrics.TotalDurationMs += event.Duration.Milliseconds()
	sr.Metrics.TotalBytesWritten += int64(event.BytesWritten)

	if event.ChaosFailure > 0 {
		sr.Metrics.Failures++
	}
	if event.ChaosCorrupted {
		sr.Metrics.CorruptedPayloads++
	}
	if event.ChaosLatency > 0 {
		sr.Metrics.TotalLatencyAddedMs += event.ChaosLatency.Milliseconds()
	}

	reqEv := RequestEvent{
		Timestamp:        event.Timestamp,
		Method:           event.Method,
		Path:             event.Path,
		Status:           event.StatusCode,
		DurationMs:       event.Duration.Milliseconds(),
		ChaosLatencyMs:   event.ChaosLatency.Milliseconds(),
		ChaosFailureCode: event.ChaosFailure,
		ChaosCorrupted:   event.ChaosCorrupted,
	}

	if len(sr.RequestEvents) < maxRequestEvents {
		sr.RequestEvents = append(sr.RequestEvents, reqEv)
	} else {
		sr.RequestEvents[sr.head%maxRequestEvents] = reqEv
	}
	sr.head++
}

// GetReport returns a safely cloned report for a given scenario.
func (r *ReportManager) GetReport(scenario string) (ScenarioReport, bool) {
	if scenario == "" {
		scenario = constants.DefaultScenario
	}
	r.mu.RLock()
	sr, ok := r.scenarios[scenario]
	r.mu.RUnlock()

	if !ok {
		return ScenarioReport{}, false
	}

	return sr.Clone(r.state.Resolve(scenario)), true
}

// GetAllReports returns safely cloned reports for all tracked scenarios.
func (r *ReportManager) GetAllReports() []ScenarioReport {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var all []ScenarioReport
	for name, sr := range r.scenarios {
		all = append(all, sr.Clone(r.state.Resolve(name)))
	}
	return all
}

// WriteReportFile dumps the report data to disk in the specified format.
func (r *ReportManager) WriteReportFile(path string, format string) error {
	if path == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	reports := r.GetAllReports()

	if format == constants.FormatJUnit {
		// TODO: proper junit formatter if required later, fallback to json for now
	}

	b, err := json.MarshalIndent(reports, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, b, 0644)
}
