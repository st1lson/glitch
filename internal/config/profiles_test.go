package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func createTempYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadProfile_Embedded(t *testing.T) {
	// "mobile" is one of the built-in profiles.
	profile, err := LoadProfile("mobile")
	if err != nil {
		t.Fatalf("expected no error loading embedded profile 'mobile', got: %v", err)
	}

	if profile.Name != "mobile" {
		t.Errorf("expected name 'mobile', got %q", profile.Name)
	}
	if profile.Latency.Min.Duration != 300*time.Millisecond {
		t.Errorf("expected min latency 300ms, got %v", profile.Latency.Min.Duration)
	}
}

func TestLoadProfile_File(t *testing.T) {
	yamlContent := `
name: custom-flaky
description: "a custom flaky profile"
latency:
  fixed: "1s"
failure:
  rate: 40
  statuses:
    - code: 502
      rate: 10
`
	path := createTempYAML(t, yamlContent)

	profile, err := LoadProfile(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if profile.Name != "custom-flaky" {
		t.Errorf("expected 'custom-flaky', got %q", profile.Name)
	}
	if profile.Latency.Fixed.Duration != 1*time.Second {
		t.Errorf("expected fixed latency 1s, got %v", profile.Latency.Fixed.Duration)
	}
	if profile.Failure.Rate != 40 {
		t.Errorf("expected failure rate 40, got %v", profile.Failure.Rate)
	}
	if len(profile.Failure.Statuses) != 1 || profile.Failure.Statuses[0].Code != 502 {
		t.Errorf("expected 1 status (502), got %v", profile.Failure.Statuses)
	}
}

func TestApplyProfile(t *testing.T) {
	// Initial clean config
	cfg := DefaultConfig()

	// Build a mock profile to apply
	p := &Profile{
		Latency: profileLatency{
			Fixed: Duration{Duration: 2 * time.Second},
		},
		Failure: profileFailure{
			Rate: 15,
			Statuses: []StatusConfig{
				{Code: 429, Rate: 100},
			},
		},
	}

	ApplyProfile(&cfg, p)

	if cfg.Latency.Fixed != 2*time.Second {
		t.Errorf("expected 2s fixed latency, got %v", cfg.Latency.Fixed)
	}
	if cfg.Failure.Rate != 15 {
		t.Errorf("expected 15 failure rate, got %v", cfg.Failure.Rate)
	}
	if len(cfg.Failure.Statuses) != 1 || cfg.Failure.Statuses[0].Code != 429 {
		t.Errorf("expected status 429 override, got %v", cfg.Failure.Statuses)
	}
}
