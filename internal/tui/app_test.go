package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/st1lson/glitch/internal/config"
	"github.com/st1lson/glitch/internal/logging"
)

func TestNewApp(t *testing.T) {
	state := config.NewManager(config.Config{})
	eventChan := make(chan logging.LogEvent, 1)
	app := New(state, eventChan)
	if app == nil {
		t.Fatal("expected non-nil App")
	}
}

func TestModel_Init(t *testing.T) {
	state := config.NewManager(config.Config{})
	m := NewModel(state)
	cmd := m.Init()
	if cmd != nil {
		t.Error("expected Init() to return nil")
	}
}

func TestModel_Update_WindowSize(t *testing.T) {
	state := config.NewManager(config.Config{})
	m := NewModel(state)

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	newModel, cmd := m.Update(msg)

	if cmd != nil {
		t.Error("expected cmd to be nil for WindowSizeMsg")
	}

	updated := newModel.(Model)
	if updated.width != 100 || updated.height != 50 {
		t.Errorf("expected width 100, height 50, got %d, %d", updated.width, updated.height)
	}
}

func TestModel_Update_KeyMsg(t *testing.T) {
	state := config.NewManager(config.Config{})
	m := NewModel(state)

	// Test Quit 'q'
	qKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	_, cmd := m.Update(qKey)
	if cmd == nil {
		t.Error("expected tea.Quit cmd for 'q'")
	}

	// Test Quit 'ctrl+c'
	ctrlCKey := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd = m.Update(ctrlCKey)
	if cmd == nil {
		t.Error("expected tea.Quit cmd for ctrl+c")
	}

	// Test Up/Down (failure rate)
	upKey := tea.KeyMsg{Type: tea.KeyUp}
	downKey := tea.KeyMsg{Type: tea.KeyDown}

	_, cmd = m.Update(upKey)
	if cmd != nil {
		t.Error("expected nil cmd for up key")
	}
	if state.Get().Failure.Rate != 5 {
		t.Errorf("expected failure rate to be 5, got %v", state.Get().Failure.Rate)
	}

	// Hit up multiple times to reach max
	for i := 0; i < 25; i++ {
		m.Update(upKey)
	}
	if state.Get().Failure.Rate != 100 {
		t.Errorf("expected max failure rate to be 100, got %v", state.Get().Failure.Rate)
	}

	_, cmd = m.Update(downKey)
	if state.Get().Failure.Rate != 95 {
		t.Errorf("expected failure rate to be 95, got %v", state.Get().Failure.Rate)
	}

	// Hit down to 0
	for i := 0; i < 25; i++ {
		m.Update(downKey)
	}
	if state.Get().Failure.Rate != 0 {
		t.Errorf("expected min failure rate to be 0, got %v", state.Get().Failure.Rate)
	}

	// Test Left/Right (profiles)
	rightKey := tea.KeyMsg{Type: tea.KeyRight}
	leftKey := tea.KeyMsg{Type: tea.KeyLeft}

	m.Update(rightKey)
	m.Update(leftKey)
}

func TestModel_Update_LogEvent(t *testing.T) {
	state := config.NewManager(config.Config{})
	m := NewModel(state)

	event := logging.LogEvent{
		StatusCode:     500,
		ChaosFailure:   500,
		ChaosLatency:   10 * time.Millisecond,
		ChaosCorrupted: true,
		Formatted:      "test log entry",
	}

	newModel, _ := m.Update(event)
	updated := newModel.(Model)

	if updated.metrics.TotalRequests != 1 {
		t.Errorf("expected 1 total request, got %d", updated.metrics.TotalRequests)
	}
	if updated.metrics.Errors != 1 {
		t.Errorf("expected 1 error, got %d", updated.metrics.Errors)
	}
	if updated.metrics.ChaosIntercepts != 1 {
		t.Errorf("expected 1 intercept, got %d", updated.metrics.ChaosIntercepts)
	}
	if len(updated.logs) != 1 || updated.logs[0] != "test log entry" {
		t.Error("expected log to be appended")
	}

	event2 := logging.LogEvent{
		StatusCode: 200,
		Formatted:  "success entry",
	}
	newModel, _ = updated.Update(event2)
	updated = newModel.(Model)

	if updated.metrics.Successes != 1 {
		t.Errorf("expected 1 success, got %d", updated.metrics.Successes)
	}

	// Test maxLogs boundary (100)
	for i := 0; i < 150; i++ {
		nm, _ := updated.Update(logging.LogEvent{Formatted: "filler"})
		updated = nm.(Model)
	}
	if len(updated.logs) != 100 {
		t.Errorf("expected exactly 100 logs, got %d", len(updated.logs))
	}
}

func TestModel_View(t *testing.T) {
	state := config.NewManager(config.Config{
		Host: "localhost",
		Port: 8080,
	})
	m := NewModel(state)

	// Test small terminal
	out := m.View()
	if !strings.Contains(out, "Terminal too small") {
		t.Errorf("expected terminal too small message, got: %s", out)
	}

	// Test proper size
	m.width = 120
	m.height = 40
	out = m.View()
	if strings.Contains(out, "Terminal too small") {
		t.Errorf("did not expect terminal too small message for 120x40")
	}
	if !strings.Contains(out, "GLITCH DASHBOARD") {
		t.Errorf("expected GLITCH DASHBOARD in output")
	}
}
