package chaos

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/st1lson/glitch/internal/config"
)

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
					Rate: 100,
				},
			},
			{
				Path:      "/api/checkout",
				Method:    "GET",
				Bandwidth: &bwOverride,
				Failure: &config.FailureConfig{
					Rate: 0,
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
			eff := evalChaos(cfg, req, nil)

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
	eff := evalChaos(cfg, req, nil)
	if eff.Bandwidth.BytesPerSecond != 102400 {
		t.Errorf("Expected 100kbps")
	}
}

func TestEvalChaos_Overrides(t *testing.T) {
	cfg := config.Config{
		Routes: []config.RouteConfig{
			{
				Path:       "*",
				Stall:      &config.StallConfig{Rate: 10},
				Corruption: &config.CorruptionConfig{Rate: 20},
				Latency:    &config.LatencyConfig{Fixed: config.Duration{}},
			},
		},
	}
	req := httptest.NewRequest("GET", "/", nil)
	eff := evalChaos(cfg, req, nil)
	if eff.Stall.Rate != 10 || eff.Corruption.Rate != 20 {
		t.Errorf("Overrides failed")
	}
}

func TestEvalChaos_RichPredicates(t *testing.T) {
	opIdx := NewOperationIndex()
	opIdx.Register("getProduct", "GET", "/products/{id}")

	cfg := config.Config{
		Failure: config.FailureConfig{Rate: 0},
		Routes: []config.RouteConfig{
			// 1. Operation ID route
			{
				OperationID: "getProduct",
				Failure:     &config.FailureConfig{Rate: 99},
			},
			// 2. Header route
			{
				Path:    "/api/*",
				Headers: map[string]string{"X-Chaos": "true"},
				Failure: &config.FailureConfig{Rate: 88},
			},
			// 3. Query route
			{
				Path:    "/search",
				Query:   map[string]string{"q": "test"},
				Failure: &config.FailureConfig{Rate: 77},
			},
			// 4. Body route
			{
				Path:   "/login",
				Method: "POST",
				Body: []config.BodyPredicate{
					{Field: "user.role", Op: "eq", Value: "guest"},
				},
				Failure: &config.FailureConfig{Rate: 66},
			},
		},
	}

	t.Run("Operation ID match", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/products/42", nil)
		eff := evalChaos(cfg, req, opIdx)
		if eff.Failure.Rate != 99 {
			t.Errorf("expected failure rate 99, got %v", eff.Failure.Rate)
		}
	})

	t.Run("Header match", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		req.Header.Set("X-Chaos", "true")
		eff := evalChaos(cfg, req, opIdx)
		if eff.Failure.Rate != 88 {
			t.Errorf("expected failure rate 88, got %v", eff.Failure.Rate)
		}
	})

	t.Run("Query match", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/search?q=test", nil)
		eff := evalChaos(cfg, req, opIdx)
		if eff.Failure.Rate != 77 {
			t.Errorf("expected failure rate 77, got %v", eff.Failure.Rate)
		}
	})

	t.Run("Body match", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"user": {"role": "guest"}}`))
		eff := evalChaos(cfg, req, opIdx)
		if eff.Failure.Rate != 66 {
			t.Errorf("expected failure rate 66, got %v", eff.Failure.Rate)
		}
	})

	t.Run("No rich predicate match", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/unmatched", nil)
		eff := evalChaos(cfg, req, opIdx)
		if eff.Failure.Rate != 0 {
			t.Errorf("expected global failure rate 0, got %v", eff.Failure.Rate)
		}
	})
}
