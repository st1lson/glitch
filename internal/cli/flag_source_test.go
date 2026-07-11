package cli

import (
	"testing"
	"time"

	"github.com/spf13/pflag"
)

func TestFlagSource_Load(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.Int("port", 3000, "")
	flags.String("host", "localhost", "")
	flags.String("proxy", "", "")
	flags.String("latency", "", "")
	flags.String("fail-rate", "", "")
	flags.StringSlice("status", nil, "")

	// Simulate user providing some flags
	flags.Set("port", "8080")
	flags.Set("proxy", "http://backend")
	flags.Set("latency", "1s")
	flags.Set("fail-rate", "10")

	args := []string{"target.json"}

	src := NewFlagSource(flags, args)
	cfg, err := src.Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
	// Host wasn't changed by user, so it should be zero-value in the override config
	if cfg.Host != "" {
		t.Errorf("expected host to be empty in override config, got %s", cfg.Host)
	}
	if cfg.Proxy != "http://backend" {
		t.Errorf("expected proxy http://backend, got %s", cfg.Proxy)
	}
	if cfg.File != "target.json" {
		t.Errorf("expected file target.json, got %s", cfg.File)
	}
	if cfg.Latency.Fixed.Duration != 1*time.Second {
		t.Errorf("expected latency 1s, got %v", cfg.Latency.Fixed.Duration)
	}
	if cfg.Failure.Rate != 10 {
		t.Errorf("expected failure rate 10, got %v", cfg.Failure.Rate)
	}
}
