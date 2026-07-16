package monkey

import (
	"context"
	"testing"
	"time"

	"github.com/st1lson/glitch/internal/config"
)

func TestMonkeyRun_AppliesPhases(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Monkey.Enabled = true
	cfg.Monkey.Phases = []config.MonkeyPhase{
		{
			Duration: config.Duration{Duration: 50 * time.Millisecond},
			Failure: config.FailureConfig{
				Rate: 10,
			},
		},
		{
			Duration: config.Duration{Duration: 50 * time.Millisecond},
			Failure: config.FailureConfig{
				Rate: 20,
			},
		},
	}

	state := config.NewState(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start monkey
	go Run(ctx, state)

	// Wait for first phase to be applied
	time.Sleep(10 * time.Millisecond)
	if state.Get().Failure.Rate != 10 {
		t.Errorf("Expected failure rate 10, got %f", state.Get().Failure.Rate)
	}

	// Wait for second phase to be applied
	time.Sleep(50 * time.Millisecond)
	if state.Get().Failure.Rate != 20 {
		t.Errorf("Expected failure rate 20, got %f", state.Get().Failure.Rate)
	}

	// Wait for loop back to first phase
	time.Sleep(50 * time.Millisecond)
	if state.Get().Failure.Rate != 10 {
		t.Errorf("Expected failure rate 10 after loop, got %f", state.Get().Failure.Rate)
	}
}

func TestMonkeyRun_Cancellation(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Monkey.Enabled = true
	cfg.Monkey.Phases = []config.MonkeyPhase{
		{
			Duration: config.Duration{Duration: 1 * time.Second},
			Failure: config.FailureConfig{Rate: 100},
		},
	}

	state := config.NewState(cfg)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		Run(ctx, state)
		close(done)
	}()

	// Ensure it starts
	time.Sleep(10 * time.Millisecond)
	if state.Get().Failure.Rate != 100 {
		t.Errorf("Expected failure rate 100, got %f", state.Get().Failure.Rate)
	}

	// Cancel context immediately
	cancel()

	// Ensure the Run function returns quickly
	select {
	case <-done:
		// success
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Run did not exit in time after cancellation")
	}
}

func TestMonkeyRun_DisabledAndReenabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Monkey.Enabled = false
	cfg.Monkey.Phases = []config.MonkeyPhase{
		{
			Duration: config.Duration{Duration: 50 * time.Millisecond},
			Failure: config.FailureConfig{Rate: 50},
		},
	}

	state := config.NewState(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go Run(ctx, state)

	// Give it a moment, shouldn't apply phase since disabled
	time.Sleep(10 * time.Millisecond)
	if state.Get().Failure.Rate != 0 {
		t.Errorf("Expected failure rate 0 while disabled, got %f", state.Get().Failure.Rate)
	}

	// Enable monkey dynamically
	state.Update(func(c *config.Config) {
		c.Monkey.Enabled = true
	})

	// Wait for the outer loop's 1s fallback tick to pick up the change.
	// Since the outer loop sleeps for 1s when disabled, we have to wait a bit longer in this test.
	// Actually, wait, this will take up to 1 second.
	time.Sleep(1100 * time.Millisecond)

	if state.Get().Failure.Rate != 50 {
		t.Errorf("Expected failure rate 50 after re-enabling, got %f", state.Get().Failure.Rate)
	}
}
