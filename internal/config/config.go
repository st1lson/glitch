package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Distribution represents the type of latency distribution.
type Distribution string

const (
	DistributionNormal  Distribution = "normal"
	DistributionUniform Distribution = "uniform"
)

// CorruptionStrategy represents a mutator type.
type CorruptionStrategy string

const (
	StrategyDropField   CorruptionStrategy = "drop_field"
	StrategySwapType    CorruptionStrategy = "swap_type"
	StrategyInjectNull  CorruptionStrategy = "inject_null"
	StrategyBreakSyntax CorruptionStrategy = "break_syntax"
)

// Bandwidth represents a network speed limit.
type Bandwidth struct {
	StringValue    string
	BytesPerSecond int
}

// UnmarshalYAML implements yaml.Unmarshaler for Bandwidth.
func (b *Bandwidth) UnmarshalYAML(unmarshal func(any) error) error {
	var raw string
	if err := unmarshal(&raw); err != nil {
		var rawInt int
		if err2 := unmarshal(&rawInt); err2 == nil {
			b.StringValue = strconv.Itoa(rawInt)
			b.BytesPerSecond = rawInt
			return nil
		}
		return err
	}

	parsed, err := ParseBandwidth(raw)
	if err != nil {
		return err
	}
	b.StringValue = raw
	b.BytesPerSecond = parsed
	return nil
}

// ParseBandwidthString creates a Bandwidth struct from a string.
func ParseBandwidthString(val string) (Bandwidth, error) {
	parsed, err := ParseBandwidth(val)
	if err != nil {
		return Bandwidth{}, err
	}
	return Bandwidth{
		StringValue:    val,
		BytesPerSecond: parsed,
	}, nil
}

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

	Bandwidth  Bandwidth        `yaml:"bandwidth"`
	Latency    LatencyConfig    `yaml:"latency"`
	Failure    FailureConfig    `yaml:"failure"`
	Stall      StallConfig      `yaml:"stall"`
	Corruption CorruptionConfig `yaml:"corruption"`
	Monkey     MonkeyConfig     `yaml:"monkey"`
	Realtime   RealtimeConfig   `yaml:"realtime"`

	Routes []RouteConfig `yaml:"routes"`
}

// RouteConfig allows overriding chaos settings for specific endpoints.
type RouteConfig struct {
	Path       string            `yaml:"path"`
	Method     string            `yaml:"method,omitempty"`
	Bandwidth  *Bandwidth        `yaml:"bandwidth,omitempty"`
	Latency    *LatencyConfig    `yaml:"latency,omitempty"`
	Failure    *FailureConfig    `yaml:"failure,omitempty"`
	Stall      *StallConfig      `yaml:"stall,omitempty"`
	Corruption *CorruptionConfig `yaml:"corruption,omitempty"`
	Realtime   *RealtimeConfig   `yaml:"realtime,omitempty"`
}

// MonkeyConfig controls dynamic chaos phases.
type MonkeyConfig struct {
	Enabled bool          `yaml:"enabled"`
	Phases  []MonkeyPhase `yaml:"phases"`
}

// MonkeyPhase defines the chaos settings for a specific duration.
type MonkeyPhase struct {
	Duration   Duration         `yaml:"duration"`
	Bandwidth  Bandwidth        `yaml:"bandwidth"`
	Latency    LatencyConfig    `yaml:"latency"`
	Failure    FailureConfig    `yaml:"failure"`
	Stall      StallConfig      `yaml:"stall"`
	Corruption CorruptionConfig `yaml:"corruption"`
	Realtime   RealtimeConfig   `yaml:"realtime"`
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

// CorruptionConfig controls JSON payload mutation.
type CorruptionConfig struct {
	Rate       float64              `yaml:"rate"`       // 0-100 percentage of JSON responses to corrupt
	Strategies []CorruptionStrategy `yaml:"strategies"` // which mutators to enable: "drop_field", "swap_type", "inject_null", "break_syntax"
	Multi      bool                 `yaml:"multi"`      // if true, apply multiple random mutators per response instead of just one
}

// Enabled returns true if corruption is configured.
func (c CorruptionConfig) Enabled() bool {
	return c.Rate > 0
}

// RealtimeConfig controls WebSocket and SSE chaos.
type RealtimeConfig struct {
	Latency             LatencyConfig `yaml:"latency"`
	DropRate            float64       `yaml:"drop_rate"`             // 0-100 percentage
	DisconnectRate      float64       `yaml:"disconnect_rate"`       // 0-100 percentage
	OutOfOrder          bool          `yaml:"out_of_order"`          // Whether to deliver messages out of order
	MaxBufferedMessages int           `yaml:"max_buffered_messages"` // Maximum messages to buffer for out of order delivery, default 100
}

// Enabled returns true if realtime chaos is configured.
func (r RealtimeConfig) Enabled() bool {
	return r.Latency.Enabled() || r.DropRate > 0 || r.DisconnectRate > 0 || r.OutOfOrder
}

// LatencyConfig controls latency injection.
type LatencyConfig struct {
	Fixed        Duration     `yaml:"fixed"`
	Min          Duration     `yaml:"min"`
	Max          Duration     `yaml:"max"`
	Distribution Distribution `yaml:"distribution"` // "normal" or "uniform"
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
		Stall: StallConfig{
			Mode:   StallModeDrop,
			DropAt: 50,
		},
		Corruption: CorruptionConfig{
			Strategies: []CorruptionStrategy{StrategyDropField, StrategySwapType, StrategyInjectNull, StrategyBreakSyntax},
		},
		Realtime: RealtimeConfig{
			MaxBufferedMessages: 100,
		},
	}
}

// HasChaos returns true if any chaos features are enabled.
func (c Config) HasChaos() bool {
	return c.Bandwidth.BytesPerSecond > 0 || c.Latency.Enabled() || c.Failure.Enabled() || c.Stall.Enabled() || c.Corruption.Enabled() || c.Monkey.Enabled || c.Realtime.Enabled() || len(c.Routes) > 0
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
