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
