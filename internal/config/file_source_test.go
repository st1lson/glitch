package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileSource_Load(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "glitch.yaml")

	content := `
port: 9999
latency:
  distribution: uniform
  max: 2s
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	src := NewFileSource(path)
	cfg, err := src.Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg == nil {
		t.Fatal("expected config, got nil")
	}

	if cfg.Port != 9999 {
		t.Errorf("expected port 9999, got %d", cfg.Port)
	}
	if cfg.Latency.Distribution != "uniform" {
		t.Errorf("expected uniform distribution, got %s", cfg.Latency.Distribution)
	}
}

func TestFileSource_Load_NotFound(t *testing.T) {
	// Explicit path should error if not found
	src := NewFileSource("/path/that/does/not/exist.yaml")
	_, err := src.Load()
	if err == nil {
		t.Error("expected error for explicit non-existent file")
	}

	// Implicit path (auto-discovery) should return nil, nil if not found
	// Assuming the current working directory doesn't have a glitch.yaml
	// (We can change working dir to temp dir to be safe)
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	tempWd := t.TempDir()
	os.Chdir(tempWd)

	srcImplicit := NewFileSource("")
	cfg, err := srcImplicit.Load()
	if err != nil {
		t.Errorf("expected no error for failed auto-discovery, got %v", err)
	}
	if cfg != nil {
		t.Errorf("expected nil config, got %v", cfg)
	}
}
