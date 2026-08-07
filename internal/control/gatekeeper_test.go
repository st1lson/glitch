package control

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/st1lson/glitch/internal/constants"
)

func TestGatekeeper_PauseResume(t *testing.T) {
	g := NewGatekeeper()
	scenario := "test-1"

	if g.IsPaused(scenario) {
		t.Fatal("expected not to be paused")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := g.Wait(ctx, scenario); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	g.Pause(scenario, 0)
	if !g.IsPaused(scenario) {
		t.Fatal("expected to be paused")
	}

	waitChan := make(chan error, 1)
	go func() {
		waitChan <- g.Wait(context.Background(), scenario)
	}()

	select {
	case <-waitChan:
		t.Fatal("expected to block")
	case <-time.After(50 * time.Millisecond):

	}

	g.Resume(scenario)
	if g.IsPaused(scenario) {
		t.Fatal("expected not to be paused")
	}

	select {
	case err := <-waitChan:
		if err != nil {
			t.Fatalf("expected nil error after resume, got %v", err)
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatal("expected to unblock")
	}
}

func TestGatekeeper_Timeout(t *testing.T) {
	g := NewGatekeeper()
	scenario := "test-2"

	g.Pause(scenario, 20*time.Millisecond)
	if !g.IsPaused(scenario) {
		t.Fatal("expected to be paused")
	}

	waitChan := make(chan error, 1)
	go func() {
		waitChan <- g.Wait(context.Background(), scenario)
	}()

	select {
	case err := <-waitChan:
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected timeout to resume")
	}

	if g.IsPaused(scenario) {
		t.Fatal("expected not to be paused after timeout")
	}
}

func TestGatekeeper_ContextCancellation(t *testing.T) {
	g := NewGatekeeper()
	scenario := "test-3"

	g.Pause(scenario, 0)
	ctx, cancel := context.WithCancel(context.Background())

	waitChan := make(chan error, 1)
	go func() {
		waitChan <- g.Wait(ctx, scenario)
	}()

	cancel()

	select {
	case err := <-waitChan:
		if err != context.Canceled {
			t.Fatalf("expected context.Canceled error, got %v", err)
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatal("expected Wait to return on cancel")
	}
}

func TestPauseMiddleware(t *testing.T) {
	g := NewGatekeeper()
	mw := PauseMiddleware(g)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	handler := mw(nextHandler)
	scenario := "test-scenario"

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(constants.HeaderScenario, scenario)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %v", rr.Code)
	}

	g.Pause(scenario, 0)
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set(constants.HeaderScenario, scenario)

	ctx, cancel := context.WithCancel(context.Background())
	req2 = req2.WithContext(ctx)

	rr2 := httptest.NewRecorder()

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504 Gateway Timeout, got %v", rr2.Code)
	}

	g.Resume(scenario)
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.Header.Set(constants.HeaderScenario, scenario)
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %v", rr3.Code)
	}
}
