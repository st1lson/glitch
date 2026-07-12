package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/st1lson/glitch/internal/config"
	"github.com/st1lson/glitch/internal/logging"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("6")).
			MarginBottom(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			Bold(true)

	metricsTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("5"))

	outerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8"))

	metricsBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("8")).
			Padding(0, 1)
	logBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("8")).
			Padding(0, 1)

	profileBoxStyle = lipgloss.NewStyle().
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			MarginTop(1)
)

const maxLogs = 100

type Metrics struct {
	TotalRequests   int
	Successes       int
	Errors          int
	ChaosIntercepts int
}

// Model represents the Bubbletea application state.
type Model struct {
	state   *config.State
	metrics Metrics
	logs    []string
	width   int
	height  int
}

// NewModel creates a new TUI model bound to the provided config state.
func NewModel(state *config.State) Model {
	return Model{
		state: state,
		logs:  make([]string, 0, maxLogs),
	}
}

// Init initializes the application.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles incoming terminal events and messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up":
			m.state.Update(func(cfg *config.Config) {
				if cfg.Failure.Rate < 100 {
					cfg.Failure.Rate += 5
					if cfg.Failure.Rate > 100 {
						cfg.Failure.Rate = 100
					}
				}
			})

		case "down":
			m.state.Update(func(cfg *config.Config) {
				if cfg.Failure.Rate > 0 {
					cfg.Failure.Rate -= 5
					if cfg.Failure.Rate < 0 {
						cfg.Failure.Rate = 0
					}
				}
			})

		case "right":
			cycleProfile(m.state, 1)

		case "left":
			cycleProfile(m.state, -1)
		}

	case logging.LogEvent:
		// Process log event
		m.metrics.TotalRequests++

		if msg.ChaosFailure > 0 || msg.ChaosLatency > 0 {
			m.metrics.ChaosIntercepts++
		}

		if msg.StatusCode >= 400 {
			m.metrics.Errors++
		} else {
			m.metrics.Successes++
		}

		// Append to ring buffer
		m.logs = append(m.logs, msg.Formatted)
		if len(m.logs) > maxLogs {
			m.logs = m.logs[1:]
		}
	}

	return m, nil
}

// cycleProfile loads the next or previous built-in profile.
func cycleProfile(state *config.State, dir int) {
	names := config.BuiltinProfileNames()
	if len(names) == 0 {
		return
	}

	state.Update(func(cfg *config.Config) {
		currentIdx := -1
		for i, n := range names {
			if cfg.ActiveProfile == n {
				currentIdx = i
				break
			}
		}

		newIdx := currentIdx + dir
		if newIdx >= len(names) {
			newIdx = 0
		} else if newIdx < 0 {
			newIdx = len(names) - 1
		}

		pName := names[newIdx]
		p, err := config.LoadProfile(pName)
		if err == nil {
			cfg.Latency = config.LatencyConfig{}
			cfg.Failure = config.FailureConfig{}
			config.ApplyProfile(cfg, p)
			cfg.ActiveProfile = pName
		}
	})
}

// View renders the application.
func (m Model) View() string {
	layout := CalculateLayout(m.width, m.height, fmt.Sprintf("⚡ GLITCH DASHBOARD - http://%s:%d", m.state.Get().Host, m.state.Get().Port))
	if layout.TotalWidth == 0 {
		return "Terminal too small or initializing..."
	}

	cfg := m.state.Get()
	headerStr := fmt.Sprintf("⚡ GLITCH DASHBOARD - http://%s:%d", cfg.Host, cfg.Port)
	hTitleFrame, _ := titleStyle.GetFrameSize()
	header := titleStyle.Width(m.width - 1 - hTitleFrame).Align(lipgloss.Center).Render(headerStr)

	// Render panes
	metricsView, actualMetricsHeight := renderMetricsPane(layout, m.metrics)
	profView := renderControlPanelPane(layout, cfg, actualMetricsHeight)
	logsView := renderLogsPane(layout, m.logs)

	// Assemble Left Column
	leftColumn := lipgloss.JoinVertical(lipgloss.Left, metricsView, profView)

	// Assemble Main Content inside Outer Border
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, logsView)
	dashboard := outerStyle.Render(mainContent)

	// Assemble Everything
	return lipgloss.JoinVertical(lipgloss.Left, header, dashboard)
}
