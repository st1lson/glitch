package config

import (
	"testing"
	"time"
)

func TestConfig_Merge(t *testing.T) {
	base := Config{
		Port: 3000,
		Host: "localhost",
		Latency: LatencyConfig{
			Fixed: DurationFromGo(100 * time.Millisecond),
		},
		Failure: FailureConfig{
			Rate: 5,
		},
	}

	override := &Config{
		Port: 8080,
		File: "api.json",
		Latency: LatencyConfig{
			Distribution: "normal",
			Max:          DurationFromGo(2 * time.Second),
		},
		Failure: FailureConfig{
			Rate: 50,
			Statuses: []StatusConfig{
				{Code: 500, Rate: 100},
			},
		},
	}

	base.Merge(override)

	if base.Port != 8080 {
		t.Errorf("expected port 8080, got %d", base.Port)
	}
	if base.Host != "localhost" {
		t.Errorf("expected host localhost, got %s", base.Host)
	}
	if base.File != "api.json" {
		t.Errorf("expected file api.json, got %s", base.File)
	}
	if base.Latency.Fixed.Duration != 100*time.Millisecond {
		t.Errorf("expected fixed latency 100ms, got %v", base.Latency.Fixed.Duration)
	}
	if base.Latency.Distribution != "normal" {
		t.Errorf("expected distribution normal, got %s", base.Latency.Distribution)
	}
	if base.Latency.Max.Duration != 2*time.Second {
		t.Errorf("expected max latency 2s, got %v", base.Latency.Max.Duration)
	}
	if base.Failure.Rate != 50 {
		t.Errorf("expected failure rate 50, got %v", base.Failure.Rate)
	}
	if len(base.Failure.Statuses) != 1 || base.Failure.Statuses[0].Code != 500 {
		t.Errorf("expected 1 status (500), got %v", base.Failure.Statuses)
	}
}

func TestConfig_Merge_Nil(t *testing.T) {
	base := Config{Port: 3000}
	base.Merge(nil)
	if base.Port != 3000 {
		t.Errorf("expected port 3000, got %d", base.Port)
	}
}

func TestConfig_Merge_ServerSettings(t *testing.T) {
	seed := int64(42)

	base := Config{Port: 3000, ControlToken: "old"}
	base.Merge(&Config{
		ControlToken:       "secret",
		InsecureControlAPI: true,
		Seed:               &seed,
		ReportPath:         "out.json",
		ReportFormat:       "junit",
	})

	if base.ControlToken != "secret" {
		t.Errorf("expected control token secret, got %q", base.ControlToken)
	}
	if !base.InsecureControlAPI {
		t.Error("expected insecure control api to be enabled")
	}
	if base.Seed == nil || *base.Seed != 42 {
		t.Errorf("expected seed 42, got %v", base.Seed)
	}
	if base.ReportPath != "out.json" {
		t.Errorf("expected report path out.json, got %q", base.ReportPath)
	}
	if base.ReportFormat != "junit" {
		t.Errorf("expected report format junit, got %q", base.ReportFormat)
	}
}

func TestConfig_Merge_KeepsServerSettingsWhenUnset(t *testing.T) {
	seed := int64(7)

	base := Config{
		ControlToken: "secret",
		Seed:         &seed,
		ReportPath:   "out.json",
		ReportFormat: "json",
	}
	base.Merge(&Config{Port: 8080})

	if base.ControlToken != "secret" {
		t.Errorf("expected control token to survive, got %q", base.ControlToken)
	}
	if base.Seed == nil || *base.Seed != 7 {
		t.Errorf("expected seed to survive, got %v", base.Seed)
	}
	if base.ReportPath != "out.json" || base.ReportFormat != "json" {
		t.Errorf("expected report settings to survive, got %q %q", base.ReportPath, base.ReportFormat)
	}
}

// A scenario overlay arrives through the control API, so it must not be able to
// change the token that guards it.
func TestConfig_MergeChaosOnly_IgnoresServerSettings(t *testing.T) {
	base := Config{ControlToken: "secret", Port: 3000}
	base.MergeChaosOnly(&Config{
		ControlToken:       "attacker",
		InsecureControlAPI: true,
		ReportPath:         "elsewhere.json",
		Port:               9999,
	})

	if base.ControlToken != "secret" {
		t.Errorf("control token was overwritten: %q", base.ControlToken)
	}
	if base.InsecureControlAPI {
		t.Error("insecure control api was enabled through a scenario overlay")
	}
	if base.ReportPath != "" {
		t.Errorf("report path was set through a scenario overlay: %q", base.ReportPath)
	}
	if base.Port != 3000 {
		t.Errorf("port was changed through a scenario overlay: %d", base.Port)
	}
}
