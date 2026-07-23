package chaos

import (
	"net/http/httptest"
	"testing"

	"github.com/st1lson/glitch/internal/config"
)

func TestMatchPath(t *testing.T) {
	tests := []struct {
		pattern  string
		path     string
		expected bool
		score    int
	}{
		{"/api/checkout", "/api/checkout", true, 1013},
		{"/api/checkout", "/api/products", false, 0},
		{"/api/*", "/api/checkout", true, 5}, // len("/api/") == 5
		{"*", "/api/checkout", true, 0},      // len("") == 0
		{"/api/products/*", "/api/products/123", true, 14}, // len("/api/products/") == 14
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			matched, score := matchPath(tt.pattern, tt.path)
			if matched != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, matched)
			}
			if score != tt.score {
				t.Errorf("expected score %v, got %v", tt.score, score)
			}
		})
	}
}

func TestEvalChaos(t *testing.T) {
	bwGlobal := config.Bandwidth{StringValue: "100kbps", BytesPerSecond: 102400}
	bwOverride := config.Bandwidth{StringValue: "50kbps", BytesPerSecond: 51200}

	cfg := config.Config{
		Bandwidth: bwGlobal,
		Failure: config.FailureConfig{
			Rate: 50,
		},
		Routes: []config.RouteConfig{
			{
				Path: "*",
				Failure: &config.FailureConfig{
					Rate: 10,
				},
			},
			{
				Path: "/api/products/*",
				Failure: &config.FailureConfig{
					Rate: 20,
				},
			},
			{
				Path:   "/api/checkout",
				Method: "POST",
				Failure: &config.FailureConfig{
					Rate: 100, // Very specific
				},
			},
			{
				Path:   "/api/checkout",
				Method: "GET",
				Bandwidth: &bwOverride,
				Failure: &config.FailureConfig{
					Rate: 0, // Specifically disabled
				},
			},
		},
	}

	tests := []struct {
		name         string
		method       string
		path         string
		expectedFail float64
		expectedBw   config.Bandwidth
	}{
		{
			name:         "Fallback to lowest wildcard",
			method:       "GET",
			path:         "/other",
			expectedFail: 10,
			expectedBw:   bwGlobal,
		},
		{
			name:         "Match more specific wildcard",
			method:       "GET",
			path:         "/api/products/1",
			expectedFail: 20,
			expectedBw:   bwGlobal,
		},
		{
			name:         "Match exact path and method POST",
			method:       "POST",
			path:         "/api/checkout",
			expectedFail: 100,
			expectedBw:   bwGlobal,
		},
		{
			name:         "Match exact path and method GET",
			method:       "GET",
			path:         "/api/checkout",
			expectedFail: 0,
			expectedBw:   bwOverride,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			eff := evalChaos(cfg, req)

			if eff.Failure.Rate != tt.expectedFail {
				t.Errorf("expected failure rate %v, got %v", tt.expectedFail, eff.Failure.Rate)
			}
			if eff.Bandwidth != tt.expectedBw {
				t.Errorf("expected bandwidth %v, got %v", tt.expectedBw, eff.Bandwidth)
			}
		})
	}
}


func TestEvalChaos_EmptyRoutes(t *testing.T) {
	cfg := config.Config{
		Bandwidth: config.Bandwidth{StringValue: "100kbps", BytesPerSecond: 102400},
	}
	req := httptest.NewRequest("GET", "/", nil)
	eff := evalChaos(cfg, req)
	if eff.Bandwidth.BytesPerSecond != 102400 {
		t.Errorf("Expected 100kbps")
	}
}

func TestEvalChaos_Overrides(t *testing.T) {
	cfg := config.Config{
		Routes: []config.RouteConfig{
			{
				Path: "*",
				Stall: &config.StallConfig{Rate: 10},
				Corruption: &config.CorruptionConfig{Rate: 20},
				Latency: &config.LatencyConfig{Fixed: config.Duration{}},
			},
		},
	}
	req := httptest.NewRequest("GET", "/", nil)
	eff := evalChaos(cfg, req)
	if eff.Stall.Rate != 10 || eff.Corruption.Rate != 20 {
		t.Errorf("Overrides failed")
	}
}
