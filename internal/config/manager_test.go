package config

import (
	"sync"
	"testing"
)

func TestManager_GetUpdate(t *testing.T) {
	initialCfg := Config{Bandwidth: Bandwidth{StringValue: "1mbps", BytesPerSecond: 1048576}}
	manager := NewManager(initialCfg)

	cfg := manager.Get()
	if cfg.Bandwidth.StringValue != "1mbps" {
		t.Errorf("expected 1mbps, got %s", cfg.Bandwidth.StringValue)
	}

	manager.Update(func(c *Config) {
		c.Bandwidth = Bandwidth{StringValue: "2mbps", BytesPerSecond: 2097152}
	})

	cfg2 := manager.Get()
	if cfg2.Bandwidth.StringValue != "2mbps" {
		t.Errorf("expected 2mbps, got %s", cfg2.Bandwidth.StringValue)
	}
}

func TestManager_Resolve(t *testing.T) {
	initialCfg := Config{Bandwidth: Bandwidth{StringValue: "1mbps", BytesPerSecond: 1048576}}
	manager := NewManager(initialCfg)

	cfg := manager.Resolve("test-scenario")
	if cfg.Bandwidth.StringValue != "1mbps" {
		t.Errorf("expected 1mbps, got %s", cfg.Bandwidth.StringValue)
	}

	manager.Overlay("test-scenario", &Config{Bandwidth: Bandwidth{StringValue: "2mbps", BytesPerSecond: 2097152}})

	cfg2 := manager.Resolve("test-scenario")
	if cfg2.Bandwidth.StringValue != "2mbps" {
		t.Errorf("expected 2mbps, got %s", cfg2.Bandwidth.StringValue)
	}

	cfg3 := manager.Baseline()
	if cfg3.Bandwidth.StringValue != "1mbps" {
		t.Errorf("expected 1mbps, got %s", cfg3.Bandwidth.StringValue)
	}
}

func TestManager_Concurrent(t *testing.T) {
	manager := NewManager(Config{Bandwidth: Bandwidth{StringValue: "init"}})
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = manager.Get()
			}
		}()
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(val Bandwidth) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				manager.Update(func(c *Config) {
					c.Bandwidth = val
				})
			}
		}(Bandwidth{StringValue: "test"})
	}

	wg.Wait()
}

func TestManager_ScenarioIsolation(t *testing.T) {
	initialCfg := Config{Bandwidth: Bandwidth{StringValue: "1mbps", BytesPerSecond: 1048576}}
	manager := NewManager(initialCfg)

	manager.Overlay("scenario-a", &Config{Bandwidth: Bandwidth{StringValue: "2mbps", BytesPerSecond: 2097152}})

	manager.Overlay("scenario-b", &Config{Bandwidth: Bandwidth{StringValue: "5mbps", BytesPerSecond: 5242880}})

	cfgA := manager.Resolve("scenario-a")
	if cfgA.Bandwidth.StringValue != "2mbps" {
		t.Errorf("scenario-a expected 2mbps, got %s", cfgA.Bandwidth.StringValue)
	}

	cfgB := manager.Resolve("scenario-b")
	if cfgB.Bandwidth.StringValue != "5mbps" {
		t.Errorf("scenario-b expected 5mbps, got %s", cfgB.Bandwidth.StringValue)
	}

	cfgDefault := manager.Resolve("")
	if cfgDefault.Bandwidth.StringValue != "1mbps" {
		t.Errorf("default scenario expected baseline 1mbps, got %s", cfgDefault.Bandwidth.StringValue)
	}

	cfgBaseline := manager.Baseline()
	if cfgBaseline.Bandwidth.StringValue != "1mbps" {
		t.Errorf("baseline expected 1mbps, got %s", cfgBaseline.Bandwidth.StringValue)
	}
}
