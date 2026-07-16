package config

import (
	"sync"
)

// State is a thread-safe wrapper around Config.
// It allows the TUI to dynamically mutate chaos settings while the HTTP engine
// safely reads from it concurrently.
type State struct {
	mu  sync.RWMutex
	cfg Config
}

// NewState initializes a thread-safe state wrapper.
func NewState(initial Config) *State {
	return &State{
		cfg: initial,
	}
}

func (s *State) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *State) Update(fn func(cfg *Config)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.cfg)
}
