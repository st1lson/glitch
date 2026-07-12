package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/st1lson/glitch/internal/config"
)

// renderRow is a helper to consistently format key-value lines across the UI
func renderRow(label string, value string, labelWidth int) string {
	var sb strings.Builder
	// Left-align the label to labelWidth
	format := fmt.Sprintf("%%-%ds", labelWidth)
	sb.WriteString(labelStyle.Render(fmt.Sprintf(format, label)))
	sb.WriteString(valueStyle.Render(value))
	sb.WriteString("\n")
	return sb.String()
}

// renderMetricsPane generates the metrics view block
func renderMetricsPane(layout Layout, metrics Metrics) (string, int) {
	var b strings.Builder
	b.WriteString(metricsTitleStyle.Render("📊 Metrics"))
	b.WriteString("\n\n")

	b.WriteString(renderRow("Total Requests:", fmt.Sprintf("%d", metrics.TotalRequests), 18))
	b.WriteString(renderRow("Successes (2xx):", fmt.Sprintf("%d", metrics.Successes), 18))
	b.WriteString(renderRow("Failures (4xx+):", fmt.Sprintf("%d", metrics.Errors), 18))
	b.WriteString(renderRow("Chaos Intercepts:", fmt.Sprintf("%d", metrics.ChaosIntercepts), 18))

	// Set the total outer height of metrics box to 10
	metricsHeight := 10
	hMetFrame, vMetFrame := metricsBoxStyle.GetFrameSize()
	
	view := metricsBoxStyle.
		Width(layout.LeftWidth - hMetFrame).
		Height(metricsHeight - vMetFrame).
		Align(lipgloss.Left).
		Render(b.String())

	return view, lipgloss.Height(view)
}

// renderControlPanelPane generates the control panel configuration block
func renderControlPanelPane(layout Layout, cfg config.Config, actualMetricsHeight int) string {
	var b strings.Builder
	b.WriteString(metricsTitleStyle.Render("🎛️  Control Panel"))
	b.WriteString("\n\n")

	profStr := "Custom"
	if cfg.ActiveProfile != "" {
		profStr = cfg.ActiveProfile
	}
	b.WriteString(renderRow("Active Profile:", profStr+"  (←/→)", 16))
	b.WriteString(renderRow("Failure Rate:", fmt.Sprintf("%.0f%%  (↑/↓)", cfg.Failure.Rate), 16))

	latStr := "0ms (Disabled)"
	if cfg.Latency.Enabled() {
		if cfg.Latency.Distribution != "" {
			latStr = fmt.Sprintf("%s (%s-%s)", cfg.Latency.Distribution, cfg.Latency.Min.Duration, cfg.Latency.Max.Duration)
		} else if cfg.Latency.Min.Duration > 0 && cfg.Latency.Max.Duration > 0 {
			latStr = fmt.Sprintf("uniform (%s-%s)", cfg.Latency.Min.Duration, cfg.Latency.Max.Duration)
		} else {
			latStr = fmt.Sprintf("fixed (%s)", cfg.Latency.Fixed.Duration)
		}
	}
	b.WriteString(renderRow("Latency Config:", latStr, 16))

	bwStr := "Unlimited"
	if cfg.Bandwidth != "" {
		bwStr = cfg.Bandwidth
	}
	b.WriteString(renderRow("Bandwidth Limit:", bwStr, 16))
	b.WriteString("\n")

	b.WriteString(helpStyle.Render("q/ctrl+c to quit"))

	// The total height of left column is contentHeight.
	profHeight := max(layout.ContentHeight-actualMetricsHeight, 5)
	
	hProfFrame, vProfFrame := profileBoxStyle.GetFrameSize()
	return profileBoxStyle.
		Width(layout.LeftWidth - hProfFrame).
		Height(profHeight - vProfFrame).
		Render(b.String())
}

// renderLogsPane generates the live request logs block
func renderLogsPane(layout Layout, logs []string) string {
	var b strings.Builder
	b.WriteString(metricsTitleStyle.Render("📝 Live Request Log"))
	b.WriteString("\n\n")

	hLogFrame, vLogFrame := logBoxStyle.GetFrameSize()
	targetRightHeight := layout.ContentHeight - vLogFrame

	// How many logs can we fit?
	visibleLogs := targetRightHeight - 2
	if visibleLogs < 1 {
		visibleLogs = 1
	}

	displayLogs := logs
	if len(logs) > visibleLogs {
		displayLogs = logs[len(logs)-visibleLogs:]
	}

	if len(displayLogs) == 0 {
		b.WriteString(labelStyle.Render("Waiting for requests..."))
	} else {
		for i, l := range displayLogs {
			b.WriteString(l)
			if i < len(displayLogs)-1 {
				b.WriteString("\n")
			}
		}
	}

	return logBoxStyle.
		Width(layout.RightWidth - hLogFrame).
		Height(targetRightHeight).
		Render(b.String())
}
