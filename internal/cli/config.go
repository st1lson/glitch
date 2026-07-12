package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/st1lson/glitch/internal/config"
)

// buildConfig orchestrates the configuration loading pipeline.
// It merges defaults, the glitch.yaml config file, chaos profiles, and CLI flags
// in a strict precedence order using a Chain of Responsibility pattern.
func buildConfig(cmd *cobra.Command, args []string) (config.Config, error) {
	flags := cmd.Flags()

	// Identify explicit config or profile paths from flags
	configPath, _ := flags.GetString("config")
	profileName, _ := flags.GetString("profile")

	// Initialize the Builder pipeline
	builder := config.NewBuilder()

	// Add Sources in order of increasing precedence
	builder.AddSource(config.NewFileSource(configPath))
	builder.AddSource(config.NewProfileSource(profileName))
	builder.AddSource(NewFlagSource(flags, args))

	// Build and deeply merge the configuration
	cfg, err := builder.Build()
	if err != nil {
		return cfg, err
	}

	// Final validation: Must have a File or Proxy defined
	if cfg.File == "" && cfg.Proxy == "" {
		return cfg, fmt.Errorf("must provide either a target file or a --proxy url")
	}

	return cfg, nil
}
