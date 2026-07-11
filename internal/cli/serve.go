package cli

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/st1lson/glitch/internal/config"
	"github.com/st1lson/glitch/internal/engine"
	"github.com/st1lson/glitch/internal/server"
)

// runServe is the main entrypoint for the serve command.
func runServe(cmd *cobra.Command, args []string) error {
	// 1. Build config
	cfg, targetFile, proxyURL, err := buildConfig(cmd, args)
	if err != nil {
		return err
	}

	// 2. Instantiate strategy Engine
	eng, err := engine.New(targetFile, proxyURL, cfg.ReadOnly)
	if err != nil {
		return err
	}

	// 3. Build router
	router := server.NewRouter(cfg, eng.Handler())

	// 4. Print startup banner
	printBanner(cfg, eng.Name(), eng.Resources())

	// 5. Start server and wait for shutdown
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	srv := server.New(addr, router)
	
	return srv.StartAndWait()
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
		return fmt.Sprintf("%s(%s,%s)", l.Distribution, l.Min, l.Max)
	}
	if l.Min > 0 && l.Max > 0 {
		return fmt.Sprintf("%s-%s", l.Min, l.Max)
	}
	return l.Fixed.String()
}
