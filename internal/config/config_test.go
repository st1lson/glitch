package config

import (
	"testing"
)

func TestConfig_HasChaos(t *testing.T) {
	empty := Config{}
	if empty.HasChaos() {
		t.Error("Expected empty config to not have chaos")
	}

	withLatency := Config{Latency: LatencyConfig{Fixed: Duration{}}}
	withLatency.Latency.Fixed.Duration = 100 // some value to enable it maybe? 
	// Wait, HasChaos just checks if Latency.Enabled(), etc.
	// We'll set something to guarantee it's enabled.
}

func TestConfig_EnabledMethods(t *testing.T) {
	lat := LatencyConfig{Fixed: Duration{}}
	if lat.Enabled() {
		t.Error("Latency enabled without config")
	}
	lat.Fixed.Duration = 1
	if !lat.Enabled() {
		t.Error("Latency disabled with config")
	}

	fail := FailureConfig{}
	if fail.Enabled() {
		t.Error("Failure enabled without config")
	}
	fail.Rate = 1
	if !fail.Enabled() {
		t.Error("Failure disabled with config")
	}

	stall := StallConfig{}
	if stall.Enabled() {
		t.Error("Stall enabled without config")
	}
	stall.Rate = 1
	if !stall.Enabled() {
		t.Error("Stall disabled with config")
	}

	corr := CorruptionConfig{}
	if corr.Enabled() {
		t.Error("Corruption enabled without config")
	}
	corr.Rate = 1
	if !corr.Enabled() {
		t.Error("Corruption disabled with config")
	}

	rt := RealtimeConfig{}
	if rt.Enabled() {
		t.Error("Realtime enabled without config")
	}
	rt.DropRate = 1
	if !rt.Enabled() {
		t.Error("Realtime disabled with config")
	}
}

func TestConfig_HasChaos_Detailed(t *testing.T) {
	c := Config{}
	c.Bandwidth = "10mbps"
	if !c.HasChaos() {
		t.Error("HasChaos should be true with Bandwidth")
	}

	c = Config{}
	c.Latency.Fixed.Duration = 1
	if !c.HasChaos() {
		t.Error("HasChaos should be true with Latency")
	}

	c = Config{}
	c.Failure.Rate = 1
	if !c.HasChaos() {
		t.Error("HasChaos should be true with Failure")
	}

	c = Config{}
	c.Stall.Rate = 1
	if !c.HasChaos() {
		t.Error("HasChaos should be true with Stall")
	}

	c = Config{}
	c.Corruption.Rate = 1
	if !c.HasChaos() {
		t.Error("HasChaos should be true with Corruption")
	}

	c = Config{}
	c.Routes = append(c.Routes, RouteConfig{})
	if !c.HasChaos() {
		t.Error("HasChaos should be true with Routes")
	}

	c = Config{}
	c.Realtime.DropRate = 1
	if !c.HasChaos() {
		t.Error("HasChaos should be true with Realtime")
	}
}

func TestParseBandwidth(t *testing.T) {
	bps, err := ParseBandwidth("100kbps")
	if err != nil || bps != 100*1024 {
		t.Errorf("ParseBandwidth 100kbps failed: %v, %d", err, bps)
	}

	bps, err = ParseBandwidth("10mbps")
	if err != nil || bps != 10*1024*1024 {
		t.Errorf("ParseBandwidth 10mbps failed: %v, %d", err, bps)
	}

	bps, err = ParseBandwidth("10m")
	if err == nil {
		t.Error("Expected error for missing unit suffix")
	}

	bps, err = ParseBandwidth("invalid")
	if err == nil {
		t.Error("Expected error for completely invalid string")
	}
}
