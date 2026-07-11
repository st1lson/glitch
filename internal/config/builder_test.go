package config

import (
	"errors"
	"testing"
)

type mockSource struct {
	cfg *Config
	err error
}

func (m *mockSource) Load() (*Config, error) {
	return m.cfg, m.err
}

func TestBuilder_Build(t *testing.T) {
	builder := NewBuilder()

	builder.AddSource(&mockSource{
		cfg: &Config{Port: 8080, Host: "0.0.0.0"},
	})

	builder.AddSource(&mockSource{
		cfg: &Config{Port: 9090}, // Should override 8080
	})

	cfg, err := builder.Build()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Port)
	}
	if cfg.Host != "0.0.0.0" {
		t.Errorf("expected host 0.0.0.0, got %s", cfg.Host)
	}
}

func TestBuilder_Build_Error(t *testing.T) {
	builder := NewBuilder()

	expectedErr := errors.New("source failed")
	builder.AddSource(&mockSource{
		err: expectedErr,
	})

	_, err := builder.Build()
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}
