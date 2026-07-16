package failure

import (
	"testing"

	"github.com/st1lson/glitch/internal/config"
)

func TestShouldTrigger(t *testing.T) {
	tests := []struct {
		name         string
		cfg          config.FailureConfig
		wantTrigger  bool
		wantCode     int
	}{
		{
			name: "No failure configured",
			cfg: config.FailureConfig{
				Rate: 0,
			},
			wantTrigger: false,
			wantCode:    0,
		},
		{
			name: "100 percent general rate",
			cfg: config.FailureConfig{
				Rate: 100,
			},
			wantTrigger: true,
			wantCode:    500,
		},
		{
			name: "100 percent specific status",
			cfg: config.FailureConfig{
				Statuses: []config.StatusConfig{
					{Code: 404, Rate: 100},
				},
			},
			wantTrigger: true,
			wantCode:    404,
		},
		{
			name: "Specific status takes precedence over general rate",
			cfg: config.FailureConfig{
				Rate: 100,
				Statuses: []config.StatusConfig{
					{Code: 503, Rate: 100},
				},
			},
			wantTrigger: true,
			wantCode:    503,
		},
		{
			name: "0 percent rates should not trigger",
			cfg: config.FailureConfig{
				Rate: 0,
				Statuses: []config.StatusConfig{
					{Code: 500, Rate: 0},
				},
			},
			wantTrigger: false,
			wantCode:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTrigger, gotCode := ShouldTrigger(tt.cfg)
			if gotTrigger != tt.wantTrigger {
				t.Errorf("ShouldTrigger() gotTrigger = %v, want %v", gotTrigger, tt.wantTrigger)
			}
			if gotCode != tt.wantCode {
				t.Errorf("ShouldTrigger() gotCode = %v, want %v", gotCode, tt.wantCode)
			}
		})
	}
}
