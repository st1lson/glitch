package config

import (
	"sync"
	"testing"
)

func TestState_GetUpdate(t *testing.T) {
	initialCfg := Config{Bandwidth: Bandwidth{StringValue: "1mbps", BytesPerSecond: 1048576}}
	state := NewState(initialCfg)

	// Get initial
	cfg := state.Get()
	if cfg.Bandwidth.StringValue != "1mbps" {
		t.Errorf("expected 1mbps, got %s", cfg.Bandwidth.StringValue)
	}

	// Update
	state.Update(func(c *Config) {
		c.Bandwidth = Bandwidth{StringValue: "2mbps", BytesPerSecond: 2097152}
	})

	cfg2 := state.Get()
	if cfg2.Bandwidth.StringValue != "2mbps" {
		t.Errorf("expected 2mbps, got %s", cfg2.Bandwidth.StringValue)
	}
}

func TestState_Concurrent(t *testing.T) {
	state := NewState(Config{Bandwidth: Bandwidth{StringValue: "init"}})
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
		go func(val Bandwidth) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				state.Update(func(c *Config) {
					c.Bandwidth = val
				})
			}
		}(Bandwidth{StringValue: "test"})
	}

	wg.Wait()
}
