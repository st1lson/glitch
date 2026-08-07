package reporting

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/st1lson/glitch/internal/config"
	"github.com/st1lson/glitch/internal/constants"
	"github.com/st1lson/glitch/internal/logging"
)

func TestNewReportManager(t *testing.T) {
	state := config.NewManager(config.Config{})
	rm := NewReportManager(state)
	if rm == nil {
		t.Fatal("expected non-nil ReportManager")
	}
}

func TestReportManager_Report_And_GetReport(t *testing.T) {
	state := config.NewManager(config.Config{})
	state.Overlay("test_scenario", &config.Config{})
	rm := NewReportManager(state)

	event := logging.LogEvent{
		Scenario:       "test_scenario",
		Timestamp:      time.Now(),
		Method:         "GET",
		Path:           "/test",
		StatusCode:     200,
		Duration:       10 * time.Millisecond,
		BytesWritten:   100,
		ChaosFailure:   500,
		ChaosCorrupted: true,
		ChaosLatency:   5 * time.Millisecond,
	}

	rm.Report(event)

	report, ok := rm.GetReport("test_scenario")
	if !ok {
		t.Fatal("expected to find report for test_scenario")
	}
	if report.Scenario != "test_scenario" {
		t.Errorf("expected scenario 'test_scenario', got %s", report.Scenario)
	}

	if report.Metrics.Requests != 1 {
		t.Errorf("expected 1 request, got %d", report.Metrics.Requests)
	}
	if report.Metrics.TotalDurationMs != 10 {
		t.Errorf("expected 10 duration, got %d", report.Metrics.TotalDurationMs)
	}
	if report.Metrics.TotalBytesWritten != 100 {
		t.Errorf("expected 100 bytes, got %d", report.Metrics.TotalBytesWritten)
	}
	if report.Metrics.Failures != 1 {
		t.Errorf("expected 1 failure, got %d", report.Metrics.Failures)
	}
	if report.Metrics.CorruptedPayloads != 1 {
		t.Errorf("expected 1 corrupted, got %d", report.Metrics.CorruptedPayloads)
	}
	if report.Metrics.TotalLatencyAddedMs != 5 {
		t.Errorf("expected 5 latency added, got %d", report.Metrics.TotalLatencyAddedMs)
	}

	if len(report.RequestEvents) != 1 {
		t.Fatalf("expected 1 request event, got %d", len(report.RequestEvents))
	}
	re := report.RequestEvents[0]
	if re.Method != "GET" || re.Path != "/test" || re.Status != 200 {
		t.Errorf("unexpected request event data: %+v", re)
	}
}

func TestReportManager_Report_DefaultScenario(t *testing.T) {
	state := config.NewManager(config.Config{})
	rm := NewReportManager(state)

	// An empty scenario or an unknown scenario should fallback to constants.DefaultScenario
	rm.Report(logging.LogEvent{Scenario: ""})

	_, ok := rm.GetReport(constants.DefaultScenario)
	if !ok {
		t.Fatal("expected default scenario to be created")
	}

	// Unknown scenario that doesn't exist in config manager
	rm.Report(logging.LogEvent{Scenario: "unknown"})
	report, ok := rm.GetReport(constants.DefaultScenario)
	if !ok {
		t.Fatal("expected unknown scenario to fallback to default")
	}
	if report.Metrics.Requests != 2 {
		t.Errorf("expected 2 requests in default scenario, got %d", report.Metrics.Requests)
	}
}

func TestReportManager_GetAllReports(t *testing.T) {
	state := config.NewManager(config.Config{})
	rm := NewReportManager(state)

	state.Overlay("s1", &config.Config{})
	rm.Report(logging.LogEvent{Scenario: "s1"})

	state.Overlay("s2", &config.Config{})
	rm.Report(logging.LogEvent{Scenario: "s2"})

	reports := rm.GetAllReports()

	if len(reports) < 2 {
		t.Fatalf("expected at least 2 reports, got %d", len(reports))
	}

	foundS1 := false
	foundS2 := false
	for _, r := range reports {
		if r.Scenario == "s1" {
			foundS1 = true
		}
		if r.Scenario == "s2" {
			foundS2 = true
		}
	}
	if !foundS1 || !foundS2 {
		t.Errorf("missing scenarios in GetAllReports: s1=%v, s2=%v", foundS1, foundS2)
	}
}

func TestReportManager_WriteReportFile(t *testing.T) {
	state := config.NewManager(config.Config{})
	rm := NewReportManager(state)

	state.Overlay("s1", &config.Config{})
	rm.Report(logging.LogEvent{Scenario: "s1"})

	dir := t.TempDir()
	path := filepath.Join(dir, "reports.json")

	err := rm.WriteReportFile(path, "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	var reports []ScenarioReport
	if err := json.Unmarshal(b, &reports); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if len(reports) == 0 {
		t.Fatal("expected reports in JSON")
	}
}

func TestReportManager_WriteReportFile_EmptyPath(t *testing.T) {
	state := config.NewManager(config.Config{})
	rm := NewReportManager(state)
	err := rm.WriteReportFile("", "json")
	if err != nil {
		t.Errorf("expected no error for empty path, got %v", err)
	}
}

func TestScenarioReport_Clone_RingBuffer(t *testing.T) {
	state := config.NewManager(config.Config{})
	rm := NewReportManager(state)
	state.Overlay("rb", &config.Config{})

	for i := 0; i < maxRequestEvents+50; i++ {
		rm.Report(logging.LogEvent{Scenario: "rb", StatusCode: i})
	}

	report, ok := rm.GetReport("rb")
	if !ok {
		t.Fatal("expected report")
	}

	if len(report.RequestEvents) != maxRequestEvents {
		t.Fatalf("expected exactly %d request events, got %d", maxRequestEvents, len(report.RequestEvents))
	}

	// The first event should be the 50th one (0-indexed 50)
	if report.RequestEvents[0].Status != 50 {
		t.Errorf("expected first event status to be 50, got %d", report.RequestEvents[0].Status)
	}
	// The last event should be maxRequestEvents + 50 - 1
	if report.RequestEvents[maxRequestEvents-1].Status != maxRequestEvents+50-1 {
		t.Errorf("expected last event status to be %d, got %d", maxRequestEvents+50-1, report.RequestEvents[maxRequestEvents-1].Status)
	}
}

func TestReportManager_GetReport_EmptyString(t *testing.T) {
	state := config.NewManager(config.Config{})
	rm := NewReportManager(state)
	rm.Report(logging.LogEvent{Scenario: ""}) // creates default

	report, ok := rm.GetReport("")
	if !ok {
		t.Fatal("expected default report when querying empty string")
	}
	if report.Scenario != constants.DefaultScenario {
		t.Errorf("expected scenario %q, got %q", constants.DefaultScenario, report.Scenario)
	}
}

func TestReportManager_GetReport_NotFound(t *testing.T) {
	state := config.NewManager(config.Config{})
	rm := NewReportManager(state)
	_, ok := rm.GetReport("nonexistent")
	if ok {
		t.Fatal("expected not to find report")
	}
}
