package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/st1lson/glitch/internal/chaos/monkey"
	"github.com/st1lson/glitch/internal/config"
	"github.com/st1lson/glitch/internal/constants"
	"github.com/st1lson/glitch/internal/control"
	"github.com/st1lson/glitch/internal/engine"
	"github.com/st1lson/glitch/internal/logging"
	"github.com/st1lson/glitch/internal/reporting"
	"github.com/st1lson/glitch/internal/server"
	"github.com/st1lson/glitch/internal/theme"
	"github.com/st1lson/glitch/internal/tui"
)

// runServe is the main entrypoint for the serve command.
func runServe(cmd *cobra.Command, args []string) error {
	cfg, err := buildConfig(cmd, args)
	if err != nil {
		return err
	}

	eng, err := engine.New(cfg.File, cfg.Proxy, cfg.ReadOnly)
	if err != nil {
		return err
	}

	state := config.NewManager(cfg)
	gate := control.NewGatekeeper()

	workerCtx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()

	if cfg.Monkey.Enabled {
		go monkey.Run(workerCtx, state)
	}

	var p *tea.Program
	var reporters logging.MultiReporter

	reports := reporting.NewReportManager(state)
	reporters = append(reporters, reports)

	if !cfg.NoTUI {

		state.Update(func(c *config.Config) {
			c.Verbose = false
		})
		app := tui.NewModel(state)
		p = tea.NewProgram(app, tea.WithAltScreen())
		reporters = append(reporters, &tuiReporter{p: p})
	} else {

		printBanner(cfg, eng.Name(), eng.Resources())
	}

	router := server.NewRouter(state, gate, eng.Handler(), reporters, reports)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	isLoopbackBind := cfg.Host == constants.LocalhostName || cfg.Host == constants.LocalhostIPv4 || cfg.Host == constants.LocalhostIPv6

	if !isLoopbackBind && cfg.ControlToken == "" && !cfg.InsecureControlAPI {
		return fmt.Errorf("refusing to expose unauthenticated control API on %s; set --control-token or use --insecure-control-api", cfg.Host)
	}

	srv := server.New(addr, router)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	if p != nil {
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running TUI: %w", err)
		}
	} else {

		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)

		select {
		case err := <-errCh:
			if err != nil {
				return err
			}
		case <-ch:

		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = srv.Shutdown(ctx)

	if cfg.ReportPath != "" {
		_ = reports.WriteReportFile(cfg.ReportPath, cfg.ReportFormat)
	}

	return nil
}

// tuiReporter bridges the logging EventReporter to the bubbletea Program.
type tuiReporter struct {
	p *tea.Program
}

func (r *tuiReporter) Report(event logging.LogEvent) {
	if r.p != nil {
		r.p.Send(event)
	}
}

// printBanner prints the colorful startup banner.
func printBanner(cfg config.Config, modeName string, resources []string) {
	bold := theme.Primary.Bold(true)
	green := theme.Success
	yellow := theme.Warning
	white := theme.Text

	fmt.Println()
	fmt.Println(bold.Render(fmt.Sprintf("  ⚡ Glitch v%s", Version)))
	fmt.Printf("%s%s\n", white.Render("  ➜ Mode: "), green.Render(modeName))
	fmt.Printf("%s%s\n", white.Render("  ➜ Server running at "), green.Render(fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port)))
	fmt.Println()

	if len(resources) > 0 {
		fmt.Println(white.Render("  Resources:"))
		for _, c := range resources {
			fmt.Println(green.Render(fmt.Sprintf("    %s", c)))
		}
		fmt.Println()
	}

	if cfg.HasChaos() {
		fmt.Println(white.Render("  Chaos:"))

		if cfg.Latency.Enabled() {
			fmt.Println(yellow.Render(fmt.Sprintf("    Latency: %s", formatLatency(cfg.Latency))))
		}

		if cfg.Failure.Enabled() {
			fmt.Println(yellow.Render(fmt.Sprintf("    Fail rate: %.0f%%", cfg.Failure.Rate)))
			if len(cfg.Failure.Statuses) > 0 {
				parts := make([]string, 0, len(cfg.Failure.Statuses))
				for _, s := range cfg.Failure.Statuses {
					parts = append(parts, fmt.Sprintf("%d:%.0f%%", s.Code, s.Rate))
				}
				fmt.Println(yellow.Render(fmt.Sprintf("    Statuses: %s", strings.Join(parts, ", "))))
			}
		}

		fmt.Println()
	}
}

// formatLatency returns a human-readable representation of the latency config.
func formatLatency(l config.LatencyConfig) string {
	if l.Distribution != "" {
		return fmt.Sprintf("%s(%s,%s)", l.Distribution, l.Min.Duration, l.Max.Duration)
	}
	if l.Min.Duration > 0 && l.Max.Duration > 0 {
		return fmt.Sprintf("%s-%s", l.Min.Duration, l.Max.Duration)
	}
	return l.Fixed.Duration.String()
}
