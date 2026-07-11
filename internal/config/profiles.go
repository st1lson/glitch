package config

import (
	"embed"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed profiles
var embeddedProfiles embed.FS

// builtinProfiles lists the profile names that are embedded in the binary.
var builtinProfiles = []string{"mobile", "3g", "bad-wifi", "production"}

// profileLatency is an intermediate struct for YAML unmarshaling of latency settings.
// It uses the custom Duration type so that human-readable strings like "300ms" are
// correctly parsed.
type profileLatency struct {
	Fixed        Duration `yaml:"fixed"`
	Min          Duration `yaml:"min"`
	Max          Duration `yaml:"max"`
	Distribution string   `yaml:"distribution"`
}

// profileFailure is an intermediate struct for YAML unmarshaling of failure settings.
type profileFailure struct {
	Rate     float64        `yaml:"rate"`
	Statuses []StatusConfig `yaml:"statuses"`
}

// Profile represents a named set of chaos-engineering settings that can be loaded
// from a YAML file and applied to a Config.
type Profile struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Latency     profileLatency `yaml:"latency"`
	Failure     profileFailure `yaml:"failure"`
}

// LoadProfile loads a chaos profile by name. It first checks the built-in
// (embedded) profiles, then falls back to treating the name as a file path.
func LoadProfile(name string) (*Profile, error) {
	// Check if it's a built-in profile name.
	for _, builtin := range builtinProfiles {
		if name == builtin {
			return loadEmbeddedProfile(name)
		}
	}

	// Fall back to loading from a file path.
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("loading profile %q: %w", name, err)
	}

	return parseProfile(data)
}

// loadEmbeddedProfile reads a profile from the embedded filesystem.
func loadEmbeddedProfile(name string) (*Profile, error) {
	data, err := embeddedProfiles.ReadFile("profiles/" + name + ".yaml")
	if err != nil {
		return nil, fmt.Errorf("reading embedded profile %q: %w", name, err)
	}
	return parseProfile(data)
}

// parseProfile unmarshals raw YAML bytes into a Profile.
func parseProfile(data []byte) (*Profile, error) {
	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing profile: %w", err)
	}
	return &p, nil
}

// ApplyProfile applies the settings from a profile to a Config.
// Only non-zero values in the profile override the existing config.
func ApplyProfile(cfg *Config, profile *Profile) {
	if profile.Latency.Fixed.Duration > 0 {
		cfg.Latency.Fixed = profile.Latency.Fixed.Duration
	}
	if profile.Latency.Min.Duration > 0 {
		cfg.Latency.Min = profile.Latency.Min.Duration
	}
	if profile.Latency.Max.Duration > 0 {
		cfg.Latency.Max = profile.Latency.Max.Duration
	}
	if profile.Latency.Distribution != "" {
		cfg.Latency.Distribution = profile.Latency.Distribution
	}

	if profile.Failure.Rate > 0 {
		cfg.Failure.Rate = profile.Failure.Rate
	}
	if len(profile.Failure.Statuses) > 0 {
		cfg.Failure.Statuses = profile.Failure.Statuses
	}
}

// BuiltinProfileNames returns the list of available built-in profile names.
func BuiltinProfileNames() []string {
	names := make([]string, len(builtinProfiles))
	copy(names, builtinProfiles)
	return names
}

// BuiltinProfileSummaries returns a formatted list of built-in profiles
// with their names and descriptions, useful for CLI help text.
func BuiltinProfileSummaries() ([]string, error) {
	summaries := make([]string, 0, len(builtinProfiles))
	for _, name := range builtinProfiles {
		p, err := loadEmbeddedProfile(name)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, fmt.Sprintf("%-12s %s", p.Name, p.Description))
	}
	return summaries, nil
}

// ProfileDuration is an alias exported for use in tests. It's equivalent to Duration.
type ProfileDuration = Duration

// DurationFromGo converts a standard time.Duration to a config.Duration.
func DurationFromGo(d time.Duration) Duration {
	return Duration{Duration: d}
}
