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

	configPath, _ := flags.GetString("config")
	profileName, _ := flags.GetString("profile")

	builder := config.NewBuilder()

	builder.AddSource(config.NewFileSource(configPath))
	builder.AddSource(config.NewProfileSource(profileName))
	builder.AddSource(NewFlagSource(flags, args))

	cfg, err := builder.Build()
	if err != nil {
		return cfg, err
	}

	if cfg.File == "" && cfg.Proxy == "" {
		return cfg, fmt.Errorf("must provide either a target file or a --proxy url")
	}

	return cfg, nil
}
