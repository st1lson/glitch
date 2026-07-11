package config

import (
	"fmt"
	"time"
)

// Duration wraps time.Duration to support YAML unmarshaling from human-readable
// strings like "300ms", "2s", "1m30s".
type Duration struct {
	time.Duration
}

// UnmarshalYAML implements yaml.Unmarshaler for Duration.
func (d *Duration) UnmarshalYAML(unmarshal func(any) error) error {
	var raw string
	if err := unmarshal(&raw); err != nil {
		return err
	}

	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", raw, err)
	}
	d.Duration = parsed
	return nil
}

// Config holds all configuration for the Glitch server.
type Config struct {
	Port     int
	Host     string
	DBFile   string
	Verbose  bool
	ReadOnly bool

	Latency LatencyConfig
	Failure FailureConfig
}

// LatencyConfig controls latency injection.
type LatencyConfig struct {
	Fixed        time.Duration `yaml:"fixed"`
	Min          time.Duration `yaml:"min"`
	Max          time.Duration `yaml:"max"`
	Distribution string        `yaml:"distribution"` // "normal" or ""
}

// Enabled returns true if any latency injection is configured.
func (l LatencyConfig) Enabled() bool {
	return l.Fixed > 0 || l.Min > 0 || l.Max > 0
}

// FailureConfig controls failure injection.
type FailureConfig struct {
	Rate     float64        `yaml:"rate"` // 0-100 percentage
	Statuses []StatusConfig `yaml:"statuses"`
}

// Enabled returns true if any failure injection is configured.
func (f FailureConfig) Enabled() bool {
	return f.Rate > 0 || len(f.Statuses) > 0
}

// StatusConfig maps a specific HTTP status code to a failure rate.
type StatusConfig struct {
	Code int     `yaml:"code"`
	Rate float64 `yaml:"rate"` // 0-100 percentage
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Port: 3000,
		Host: "localhost",
	}
}

// HasChaos returns true if any chaos features are enabled.
func (c Config) HasChaos() bool {
	return c.Latency.Enabled() || c.Failure.Enabled()
}
