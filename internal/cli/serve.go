package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/st1lson/glitch/internal/config"
	"github.com/st1lson/glitch/internal/engine"
	"github.com/st1lson/glitch/internal/logging"
	"github.com/st1lson/glitch/internal/server"
	"github.com/st1lson/glitch/internal/tui"
)

// runServe is the main entrypoint for the serve command.
func runServe(cmd *cobra.Command, args []string) error {
	// 1. Build config
	cfg, err := buildConfig(cmd, args)
	if err != nil {
		return err
	}

	// 2. Instantiate strategy Engine
	eng, err := engine.New(cfg.File, cfg.Proxy, cfg.ReadOnly)
	if err != nil {
		return err
	}

	// 3. Build thread-safe state
	state := config.NewState(cfg)

	// 4. Setup TUI Program (if enabled)
	var reporter logging.EventReporter
	var p *tea.Program

	if !cfg.NoTUI {
		// Disable verbose logging to stdout when TUI is running
		state.Update(func(c *config.Config) {
			c.Verbose = false
		})
		app := tui.NewModel(state)
		p = tea.NewProgram(app, tea.WithAltScreen()) // WithAltScreen is nice for dashboards
		reporter = &tuiReporter{p: p}
	} else {
		// Print standard startup banner when not using TUI
		printBanner(cfg, eng.Name(), eng.Resources())
	}

	// 5. Build router with the reporter
	router := server.NewRouter(state, eng.Handler(), reporter)

	// 6. Start HTTP server in the background
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	srv := server.New(addr, router)
	
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	// 7. Run the TUI (if enabled)
	if p != nil {
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running TUI: %w", err)
		}
	} else {
		// If TUI is disabled, wait for OS signals manually to gracefully shutdown
		// Actually, startAndWait already did this, but since we are running in background,
		// we just wait for a signal here or just use srv.StartAndWait().
		// For simplicity, just wait on the error channel (which blocks forever until error or exit)
		// Better: block here until we get an interrupt, but the router setup is already done.
		// Actually, since we replaced srv.StartAndWait() which handled sigint, we should just
		// listen for interrupt.
		
		// For now, if no-tui, let's just block on errCh. 
		// Actually, let's just restore blocking wait if no TUI!
		
		// If TUI is disabled, wait for OS signals manually to gracefully shutdown
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)

		select {
		case err := <-errCh:
			if err != nil {
				return err
			}
		case <-ch:
			// Shutting down
		}
	}

	// 8. Gracefully shut down server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	_ = srv.Shutdown(ctx)
	
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
	bold := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	white := color.New(color.FgWhite)

	fmt.Println()
	bold.Printf("  ⚡ Glitch v%s\n", Version)
	white.Print("  ➜ Mode: ")
	green.Printf("%s\n", modeName)
	white.Print("  ➜ Server running at ")
	green.Printf("http://%s:%d\n", cfg.Host, cfg.Port)
	fmt.Println()

	// Resources
	if len(resources) > 0 {
		white.Println("  Resources:")
		for _, c := range resources {
			green.Printf("    %s\n", c)
		}
		fmt.Println()
	}

	// Chaos settings
	if cfg.HasChaos() {
		white.Println("  Chaos:")

		if cfg.Latency.Enabled() {
			yellow.Printf("    Latency: %s\n", formatLatency(cfg.Latency))
		}

		if cfg.Failure.Enabled() {
			yellow.Printf("    Fail rate: %.0f%%\n", cfg.Failure.Rate)
			if len(cfg.Failure.Statuses) > 0 {
				parts := make([]string, 0, len(cfg.Failure.Statuses))
				for _, s := range cfg.Failure.Statuses {
					parts = append(parts, fmt.Sprintf("%d:%.0f%%", s.Code, s.Rate))
				}
				yellow.Printf("    Statuses: %s\n", strings.Join(parts, ", "))
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
