package config

// Merge applies non-zero fields from the override config onto the base Config.
// It deep merges nested structs like Latency and Failure.
func (c *Config) Merge(override *Config) {
	if override == nil {
		return
	}

	if override.Port != 0 {
		c.Port = override.Port
	}
	if override.Host != "" {
		c.Host = override.Host
	}
	if override.File != "" {
		c.File = override.File
	}
	if override.Proxy != "" {
		c.Proxy = override.Proxy
	}
	if override.Verbose {
		c.Verbose = true
	}
	if override.ReadOnly {
		c.ReadOnly = true
	}
	if override.NoTUI {
		c.NoTUI = true
	}
	if override.ActiveProfile != "" {
		c.ActiveProfile = override.ActiveProfile
	}
	if override.Bandwidth != "" {
		c.Bandwidth = override.Bandwidth
	}

	c.mergeLatency(&override.Latency)
	c.mergeFailure(&override.Failure)
	c.mergeCorruption(&override.Corruption)
	c.mergeStall(&override.Stall)
	c.mergeMonkey(&override.Monkey)
	c.mergeRealtime(&override.Realtime)
	if len(override.Routes) > 0 {
		c.Routes = override.Routes
	}
}

func (c *Config) mergeLatency(override *LatencyConfig) {
	if override.Fixed.Duration > 0 {
		c.Latency.Fixed = override.Fixed
	}
	if override.Min.Duration > 0 {
		c.Latency.Min = override.Min
	}
	if override.Max.Duration > 0 {
		c.Latency.Max = override.Max
	}
	if override.Distribution != "" {
		c.Latency.Distribution = override.Distribution
	}
}

func (c *Config) mergeFailure(override *FailureConfig) {
	if override.Rate > 0 {
		c.Failure.Rate = override.Rate
	}
	if len(override.Statuses) > 0 {
		c.Failure.Statuses = override.Statuses
	}
}

func (c *Config) mergeCorruption(override *CorruptionConfig) {
	if override.Rate > 0 {
		c.Corruption.Rate = override.Rate
	}
	if len(override.Strategies) > 0 {
		c.Corruption.Strategies = override.Strategies
	}
	if override.Multi {
		c.Corruption.Multi = true
	}
}

func (c *Config) mergeStall(override *StallConfig) {
	if override.Rate > 0 {
		c.Stall.Rate = override.Rate
	}
	if override.Mode != "" {
		c.Stall.Mode = override.Mode
	}
	if override.DropAt > 0 {
		c.Stall.DropAt = override.DropAt
	}
}

func (c *Config) mergeMonkey(override *MonkeyConfig) {
	if override.Enabled {
		c.Monkey.Enabled = true
	}
	if len(override.Phases) > 0 {
		c.Monkey.Phases = override.Phases
	}
}

func (c *Config) mergeRealtime(override *RealtimeConfig) {
	if override.Latency.Fixed.Duration > 0 {
		c.Realtime.Latency.Fixed = override.Latency.Fixed
	}
	if override.Latency.Min.Duration > 0 {
		c.Realtime.Latency.Min = override.Latency.Min
	}
	if override.Latency.Max.Duration > 0 {
		c.Realtime.Latency.Max = override.Latency.Max
	}
	if override.Latency.Distribution != "" {
		c.Realtime.Latency.Distribution = override.Latency.Distribution
	}
	if override.DropRate > 0 {
		c.Realtime.DropRate = override.DropRate
	}
	if override.DisconnectRate > 0 {
		c.Realtime.DisconnectRate = override.DisconnectRate
	}
	if override.OutOfOrder {
		c.Realtime.OutOfOrder = true
	}
	if override.MaxBufferedMessages > 0 {
		c.Realtime.MaxBufferedMessages = override.MaxBufferedMessages
	}
}
