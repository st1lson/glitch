package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/st1lson/glitch/internal/config"
)

// buildConfig extracts all imperative flag parsing, profile merging,
// and CLI overrides into a single builder function, returning the
// unified config object along with the targetFile and proxyURL.
func buildConfig(cmd *cobra.Command, args []string) (config.Config, string, string, error) {
	cfg := config.DefaultConfig()
	flags := cmd.Flags()

	// 1. Basic flags
	if v, err := flags.GetInt("port"); err == nil {
		cfg.Port = v
	}
	if v, err := flags.GetString("host"); err == nil && v != "" {
		cfg.Host = v
	}
	if v, err := flags.GetBool("verbose"); err == nil {
		cfg.Verbose = v
	}
	if v, err := flags.GetBool("read-only"); err == nil {
		cfg.ReadOnly = v
	}

	proxyTarget, _ := flags.GetString("proxy")
	var dbFile string
	if len(args) > 0 {
		dbFile = args[0]
		cfg.DBFile = dbFile
	}

	if proxyTarget == "" && dbFile == "" {
		return cfg, "", "", fmt.Errorf("must provide either a target file or a --proxy url")
	}

	// 2. Load and apply chaos profile
	profileName, _ := flags.GetString("profile")
	if profileName != "" {
		profile, err := config.LoadProfile(profileName)
		if err != nil {
			return cfg, "", "", fmt.Errorf("loading profile %q: %w", profileName, err)
		}
		config.ApplyProfile(&cfg, profile)
	}

	// 3. Parse latency overrides
	latencyStr, _ := flags.GetString("latency")
	if latencyStr != "" {
		latCfg, err := ParseLatency(latencyStr)
		if err != nil {
			return cfg, "", "", fmt.Errorf("parsing --latency: %w", err)
		}
		cfg.Latency = latCfg
	}

	// 4. Parse failure overrides
	failRateStr, _ := flags.GetString("fail-rate")
	if failRateStr != "" {
		rate, err := ParseFailRate(failRateStr)
		if err != nil {
			return cfg, "", "", fmt.Errorf("parsing --fail-rate: %w", err)
		}
		cfg.Failure.Rate = rate
	}

	statusVals, _ := flags.GetStringSlice("status")
	if len(statusVals) > 0 {
		statuses, err := ParseStatusFlags(statusVals)
		if err != nil {
			return cfg, "", "", fmt.Errorf("parsing --status: %w", err)
		}
		cfg.Failure.Statuses = statuses
	}

	return cfg, dbFile, proxyTarget, nil
}
