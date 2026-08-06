package rng

import (
	"context"
	"sync"
	"testing"
)

func TestDeterministicSequence(t *testing.T) {
	rng1 := New(42)
	rng2 := New(42)

	for i := 0; i < 100; i++ {
		val1 := rng1.Float64()
		val2 := rng2.Float64()
		if val1 != val2 {
			t.Fatalf("Sequence divergence at index %d: %v != %v", i, val1, val2)
		}
	}
}

func TestDifferentSeeds(t *testing.T) {
	rng1 := New(42)
	rng2 := New(43)
	
	// While it's theoretically possible for the first float to match,
	// practically it won't.
	if rng1.Float64() == rng2.Float64() {
		t.Fatal("Different seeds generated the same first value")
	}
}

func TestThreadSafety(t *testing.T) {
	rng := New(42)
	var wg sync.WaitGroup
	
	// Just verify it doesn't crash when accessed concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				_ = rng.Float64()
				_ = rng.IntN(100)
			}
		}()
	}
	wg.Wait()
}

func TestContextPropagation(t *testing.T) {
	ctx := context.Background()
	rng := New(42)
	
	ctxWithRNG := WithRNG(ctx, rng)
	extractedRNG := FromContext(ctxWithRNG)
	
	if rng != extractedRNG {
		t.Fatal("Extracted RNG pointer does not match original")
	}
}

func TestFromContextFallback(t *testing.T) {
	ctx := context.Background()
	rng := FromContext(ctx)
	
	if rng == nil {
		t.Fatal("Expected fallback RNG when context has no RNG, got nil")
	}
	
	// Should be able to use it
	_ = rng.Float64()
}
