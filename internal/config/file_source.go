package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// FileSource loads configuration from a global YAML file (e.g. glitch.yaml).
type FileSource struct {
	Path string
}

// NewFileSource returns a FileSource that will attempt to read the specified path.
func NewFileSource(path string) *FileSource {
	return &FileSource{Path: path}
}

// Load attempts to parse the configuration file. If the path is empty,
// it falls back to auto-discovering 'glitch.yaml' or '.glitch.yaml' in the
// current directory.
func (s *FileSource) Load() (*Config, error) {
	path := s.Path
	if path == "" {
		discovered, found := DiscoverConfigFile()
		if !found {
			return nil, nil // No config file found, totally fine.
		}
		path = discovered
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("config file not found: %s", path)
		}
		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %q: %w", path, err)
	}

	return &cfg, nil
}

// DiscoverConfigFile checks the current working directory for default
// configuration files, returning the path and a boolean indicating success.
func DiscoverConfigFile() (string, bool) {
	candidates := []string{"glitch.yaml", ".glitch.yaml"}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, true
		}
	}
	return "", false
}
