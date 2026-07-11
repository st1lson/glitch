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
	Port     int    `yaml:"port"`
	Host     string `yaml:"host"`
	File     string `yaml:"file"`
	Proxy    string `yaml:"proxy"`
	Verbose  bool   `yaml:"verbose"`
	ReadOnly bool   `yaml:"read_only"`

	Latency LatencyConfig `yaml:"latency"`
	Failure FailureConfig `yaml:"failure"`
}

// LatencyConfig controls latency injection.
type LatencyConfig struct {
	Fixed        Duration `yaml:"fixed"`
	Min          Duration `yaml:"min"`
	Max          Duration `yaml:"max"`
	Distribution string   `yaml:"distribution"` // "normal" or "uniform"
}

// Enabled returns true if any latency injection is configured.
func (l LatencyConfig) Enabled() bool {
	return l.Fixed.Duration > 0 || l.Min.Duration > 0 || l.Max.Duration > 0
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
