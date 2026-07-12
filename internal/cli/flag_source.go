package cli

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/st1lson/glitch/internal/config"
)

// FlagSource extracts configuration overrides directly from the CLI flags.
type FlagSource struct {
	flags *pflag.FlagSet
	args  []string
}

// NewFlagSource returns a new FlagSource.
func NewFlagSource(flags *pflag.FlagSet, args []string) *FlagSource {
	return &FlagSource{
		flags: flags,
		args:  args,
	}
}

// Load inspects the provided flags and args and returns a Config with
// only the explicitly provided values set.
func (s *FlagSource) Load() (*config.Config, error) {
	var cfg config.Config

	// Basic flags
	if s.flags.Changed("port") {
		cfg.Port, _ = s.flags.GetInt("port")
	}
	if s.flags.Changed("host") {
		cfg.Host, _ = s.flags.GetString("host")
	}
	if s.flags.Changed("verbose") {
		cfg.Verbose, _ = s.flags.GetBool("verbose")
	}
	if s.flags.Changed("read-only") {
		cfg.ReadOnly, _ = s.flags.GetBool("read-only")
	}
	if s.flags.Changed("no-tui") {
		cfg.NoTUI, _ = s.flags.GetBool("no-tui")
	}
	if s.flags.Changed("proxy") {
		cfg.Proxy, _ = s.flags.GetString("proxy")
	}

	// Positional arguments (Target file)
	if len(s.args) > 0 {
		cfg.File = s.args[0]
	}

	// Chaos Flags
	if s.flags.Changed("latency") {
		latencyStr, _ := s.flags.GetString("latency")
		latCfg, err := ParseLatency(latencyStr)
		if err != nil {
			return nil, fmt.Errorf("parsing --latency: %w", err)
		}
		cfg.Latency = latCfg
	}

	if s.flags.Changed("fail-rate") {
		failRateStr, _ := s.flags.GetString("fail-rate")
		rate, err := ParseFailRate(failRateStr)
		if err != nil {
			return nil, fmt.Errorf("parsing --fail-rate: %w", err)
		}
		cfg.Failure.Rate = rate
	}

	if s.flags.Changed("status") {
		statusVals, _ := s.flags.GetStringSlice("status")
		statuses, err := ParseStatusFlags(statusVals)
		if err != nil {
			return nil, fmt.Errorf("parsing --status: %w", err)
		}
		cfg.Failure.Statuses = statuses
	}

	return &cfg, nil
}
