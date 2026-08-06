package config

import (
	"sync"
)

// Manager is a thread-safe wrapper that manages the baseline configuration
// and any per-scenario overlays applied via the control API.
type Manager struct {
	mu        sync.RWMutex
	baseline  Config
	scenarios map[string]*Config
}

// NewManager initializes a thread-safe configuration manager.
// The provided initial config becomes the immutable baseline.
func NewManager(initial Config) *Manager {
	return &Manager{
		baseline:  initial,
		scenarios: make(map[string]*Config),
	}
}

// Baseline returns a copy of the immutable startup configuration.
func (m *Manager) Baseline() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.baseline
}

// Resolve returns the effective configuration for a given scenario.
// If the scenario has no overrides, it returns the baseline.
func (m *Manager) Resolve(scenario string) Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if cfg, ok := m.scenarios[scenario]; ok {
		return *cfg
	}
	return m.baseline
}

// Get is a convenience method that returns the effective configuration
// for the default (empty) scenario. Used by the TUI and older APIs.
func (m *Manager) Get() Config {
	return m.Resolve("")
}

// Has returns true if the specified scenario explicitly exists in the configuration.
func (m *Manager) Has(scenario string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.scenarios[scenario]
	return ok
}

// Overlay applies chaos settings on top of a scenario's current configuration.
// If the scenario doesn't exist, it is initialized from the baseline first.
func (m *Manager) Overlay(scenario string, override *Config) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg, ok := m.scenarios[scenario]
	if !ok {
		// Start from baseline
		newCfg := m.baseline
		cfg = &newCfg
		m.scenarios[scenario] = cfg
	}
	
	cfg.MergeChaosOnly(override)
}

// Reset clears all overrides for a specific scenario, returning it to the baseline.
func (m *Manager) Reset(scenario string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.scenarios, scenario)
}

// ResetAndApplyProfile clears a scenario's overrides and applies a specific profile.
func (m *Manager) ResetAndApplyProfile(scenario string, profile *Profile) {
	m.mu.Lock()
	defer m.mu.Unlock()

	newCfg := m.baseline
	ApplyProfile(&newCfg, profile)
	m.scenarios[scenario] = &newCfg
}

// Update is a convenience method that safely mutates the default scenario's configuration
// using a callback function. Used by the TUI.
func (m *Manager) Update(fn func(cfg *Config)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg, ok := m.scenarios[""]
	if !ok {
		newCfg := m.baseline
		cfg = &newCfg
		m.scenarios[""] = cfg
	}
	
	fn(cfg)
}
