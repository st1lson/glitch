package config

import (
	"fmt"
	"strconv"
	"strings"
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
	NoTUI    bool   `yaml:"no_tui"`
	
	ActiveProfile string `yaml:"-"`

	Bandwidth string        `yaml:"bandwidth"`
	Latency   LatencyConfig `yaml:"latency"`
	Failure   FailureConfig `yaml:"failure"`
	Stall     StallConfig   `yaml:"stall"`
}

// StallMode represents the type of stall injection.
type StallMode string

const (
	StallModeDrop StallMode = "drop"
	StallModeHang StallMode = "hang"
)

// StallConfig controls mid-flight network stalls and connection drops.
type StallConfig struct {
	Rate   float64   `yaml:"rate"`    // 0-100 percentage
	Mode   StallMode `yaml:"mode"`    // "drop" (TCP reset) or "hang" (block indefinitely)
	DropAt float64   `yaml:"drop_at"` // 0-100 percentage of payload to stream before stalling (default 50)
}

// Enabled returns true if stall injection is configured.
func (s StallConfig) Enabled() bool {
	return s.Rate > 0
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
	return c.Bandwidth != "" || c.Latency.Enabled() || c.Failure.Enabled() || c.Stall.Enabled()
}
// ParseBandwidth parses a bandwidth string into bytes per second.
// Supports suffixes: kbps, mbps, b/s, kb/s, mb/s.
func ParseBandwidth(val string) (int, error) {
	val = strings.TrimSpace(strings.ToLower(val))
	if val == "" || val == "0" || val == "unlimited" {
		return 0, nil
	}

	var multiplier float64 = 1
	var numStr string

	if strings.HasSuffix(val, "kbps") || strings.HasSuffix(val, "kb/s") {
		multiplier = 1024
		numStr = strings.TrimSuffix(val, "kbps")
		numStr = strings.TrimSuffix(numStr, "kb/s")
	} else if strings.HasSuffix(val, "mbps") || strings.HasSuffix(val, "mb/s") {
		multiplier = 1024 * 1024
		numStr = strings.TrimSuffix(val, "mbps")
		numStr = strings.TrimSuffix(numStr, "mb/s")
	} else if strings.HasSuffix(val, "bps") || strings.HasSuffix(val, "b/s") {
		multiplier = 1
		numStr = strings.TrimSuffix(val, "bps")
		numStr = strings.TrimSuffix(numStr, "b/s")
	} else {
		numStr = val // default to bytes if no suffix
	}

	numStr = strings.TrimSpace(numStr)
	
	rate, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid bandwidth value %q: %w", val, err)
	}

	if rate <= 0 {
		return 0, fmt.Errorf("bandwidth must be positive")
	}

	return int(rate * multiplier), nil
}
