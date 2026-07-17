package config

import (
	"sync"
	"testing"
)

func TestState_GetUpdate(t *testing.T) {
	initialCfg := Config{Bandwidth: "1mbps"}
	state := NewState(initialCfg)

	// Get initial
	cfg := state.Get()
	if cfg.Bandwidth != "1mbps" {
		t.Errorf("expected 1mbps, got %s", cfg.Bandwidth)
	}

	// Update
	state.Update(func(c *Config) {
		c.Bandwidth = "2mbps"
	})

	cfg2 := state.Get()
	if cfg2.Bandwidth != "2mbps" {
		t.Errorf("expected 2mbps, got %s", cfg2.Bandwidth)
	}
}

func TestState_Concurrent(t *testing.T) {
	state := NewState(Config{Bandwidth: "init"})
	var wg sync.WaitGroup

	// Readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = state.Get()
			}
		}()
	}

	// Writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(val string) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				state.Update(func(c *Config) {
					c.Bandwidth = val
				})
			}
		}("test")
	}

	wg.Wait()
}
