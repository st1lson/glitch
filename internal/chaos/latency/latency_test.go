package latency

import (
	"context"
	"testing"
	"time"

	"github.com/st1lson/glitch/internal/config"
)

func TestComputeDelay(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.LatencyConfig
		want time.Duration
	}{
		{
			name: "No latency configured",
			cfg:  config.LatencyConfig{},
			want: 0,
		},
		{
			name: "Fixed duration",
			cfg: config.LatencyConfig{
				Fixed: config.Duration{Duration: 50 * time.Millisecond},
			},
			want: 50 * time.Millisecond,
		},
		{
			name: "Uniform distribution exact diff 0",
			cfg: config.LatencyConfig{
				Min: config.Duration{Duration: 10 * time.Millisecond},
				Max: config.Duration{Duration: 10 * time.Millisecond},
			},
			want: 10 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeDelay(tt.cfg)
			if got != tt.want {
				t.Errorf("computeDelay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComputeDelay_Ranges(t *testing.T) {
	// Test range distributions (uniform and normal)
	cfgUniform := config.LatencyConfig{
		Min: config.Duration{Duration: 10 * time.Millisecond},
		Max: config.Duration{Duration: 50 * time.Millisecond},
	}

	cfgNormal := config.LatencyConfig{
		Min:          config.Duration{Duration: 10 * time.Millisecond},
		Max:          config.Duration{Duration: 50 * time.Millisecond},
		Distribution: "normal",
	}

	for range 100 {
		gotUniform := computeDelay(cfgUniform)
		if gotUniform < 10*time.Millisecond || gotUniform > 50*time.Millisecond {
			t.Errorf("computeDelay() uniform = %v, want between 10ms and 50ms", gotUniform)
		}

		gotNormal := computeDelay(cfgNormal)
		if gotNormal < 10*time.Millisecond || gotNormal > 50*time.Millisecond {
			t.Errorf("computeDelay() normal = %v, want between 10ms and 50ms", gotNormal)
		}
	}
}

func TestInject(t *testing.T) {
	t.Run("No delay", func(t *testing.T) {
		cfg := config.LatencyConfig{}
		start := time.Now()
		waited := Inject(context.Background(), cfg)
		if waited > 5*time.Millisecond {
			t.Errorf("Inject() without config took too long: %v", time.Since(start))
		}
	})

	t.Run("Fixed delay", func(t *testing.T) {
		cfg := config.LatencyConfig{
			Fixed: config.Duration{Duration: 20 * time.Millisecond},
		}
		start := time.Now()
		waited := Inject(context.Background(), cfg)
		elapsed := time.Since(start)

		if elapsed < 20*time.Millisecond {
			t.Errorf("Inject() elapsed %v, want >= 20ms", elapsed)
		}
		if waited < 20*time.Millisecond {
			t.Errorf("Inject() returned waited %v, want >= 20ms", waited)
		}
	})

	t.Run("Context cancelled", func(t *testing.T) {
		cfg := config.LatencyConfig{
			Fixed: config.Duration{Duration: 100 * time.Millisecond},
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		start := time.Now()
		waited := Inject(ctx, cfg)
		elapsed := time.Since(start)

		if elapsed > 50*time.Millisecond {
			t.Errorf("Inject() elapsed %v, want < 50ms because of context cancellation", elapsed)
		}
		if waited > 50*time.Millisecond {
			t.Errorf("Inject() returned waited %v, want < 50ms", waited)
		}
	})
}
