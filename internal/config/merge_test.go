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
