package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/st1lson/glitch/internal/config"
)

// ParseLatency parses a latency flag value into a LatencyConfig.
//
// Supported formats:
//
//	"2s"              → Fixed = 2s
//	"500ms-3s"        → Min = 500ms, Max = 3s
//	"normal:200ms,2s" → Distribution = "normal", Min = 200ms, Max = 2s
func ParseLatency(val string) (config.LatencyConfig, error) {
	var cfg config.LatencyConfig

	// Distribution format: "distribution:min,max"
	if idx := strings.Index(val, ":"); idx != -1 {
		dist := val[:idx]
		if dist != "normal" && dist != "uniform" {
			return cfg, fmt.Errorf("invalid latency distribution %q: must be 'normal' or 'uniform'", dist)
		}
		cfg.Distribution = dist
		rest := val[idx+1:]

		parts := strings.SplitN(rest, ",", 2)
		if len(parts) != 2 {
			return cfg, fmt.Errorf("invalid latency distribution format %q: expected \"dist:min,max\"", val)
		}

		minDur, err := time.ParseDuration(strings.TrimSpace(parts[0]))
		if err != nil {
			return cfg, fmt.Errorf("invalid latency min duration %q: %w", parts[0], err)
		}

		maxDur, err := time.ParseDuration(strings.TrimSpace(parts[1]))
		if err != nil {
			return cfg, fmt.Errorf("invalid latency max duration %q: %w", parts[1], err)
		}

		cfg.Min = config.Duration{Duration: minDur}
		cfg.Max = config.Duration{Duration: maxDur}
		return cfg, nil
	}

	// Range format: "min-max" (e.g. "500ms-3s")
	// We split on "-" but need to be careful with durations like "500ms".
	// Strategy: find the first "-" that is preceded by a letter or digit that
	// could end a duration token (s, m, h, etc.) and followed by a digit.
	if dashIdx := findRangeSeparator(val); dashIdx != -1 {
		minStr := val[:dashIdx]
		maxStr := val[dashIdx+1:]

		minDur, err := time.ParseDuration(minStr)
		if err != nil {
			return cfg, fmt.Errorf("invalid latency min duration %q: %w", minStr, err)
		}

		maxDur, err := time.ParseDuration(maxStr)
		if err != nil {
			return cfg, fmt.Errorf("invalid latency max duration %q: %w", maxStr, err)
		}

		cfg.Min = config.Duration{Duration: minDur}
		cfg.Max = config.Duration{Duration: maxDur}
		return cfg, nil
	}

	// Fixed duration: "2s"
	d, err := time.ParseDuration(val)
	if err != nil {
		return cfg, fmt.Errorf("invalid latency duration %q: %w", val, err)
	}

	cfg.Fixed = config.Duration{Duration: d}
	return cfg, nil
}

// findRangeSeparator locates the dash separating two durations (e.g. "500ms-3s").
// It returns the index of the separator dash, or -1 if none is found.
// A valid separator dash is preceded by a duration-ending letter (s, m, h, etc.)
// and followed by a digit.
func findRangeSeparator(val string) int {
	for i := 1; i < len(val)-1; i++ {
		if val[i] == '-' && isLetter(val[i-1]) && isDigit(val[i+1]) {
			return i
		}
	}
	return -1
}

func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// ParseStatusFlags parses a slice of "code:rate" strings into StatusConfig entries.
//
// Example input: ["500:10", "429:5"] → [{Code:500, Rate:10}, {Code:429, Rate:5}]
func ParseStatusFlags(vals []string) ([]config.StatusConfig, error) {
	statuses := make([]config.StatusConfig, 0, len(vals))

	for _, v := range vals {
		parts := strings.SplitN(v, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid status format %q: expected \"code:rate\"", v)
		}

		code, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("invalid status code %q: %w", parts[0], err)
		}
		if code < 100 || code >= 600 {
			return nil, fmt.Errorf("invalid status code %d: must be between 100 and 599", code)
		}

		rateStr := strings.TrimSpace(parts[1])
		rateStr = strings.TrimSuffix(rateStr, "%")
		
		rate, err := strconv.ParseFloat(rateStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid status rate %q: %w", parts[1], err)
		}

		statuses = append(statuses, config.StatusConfig{
			Code: code,
			Rate: rate,
		})
	}

	return statuses, nil
}

// ParseFailRate parses a failure rate string like "20" or "20%" into a float64.
// The returned value is the numeric percentage (e.g. 20.0), not a fraction.
func ParseFailRate(val string) (float64, error) {
	val = strings.TrimSpace(val)
	val = strings.TrimSuffix(val, "%")

	rate, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid fail rate %q: %w", val, err)
	}

	if rate < 0 || rate > 100 {
		return 0, fmt.Errorf("fail rate must be between 0 and 100, got %v", rate)
	}

	return rate, nil
}
