package control

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/st1lson/glitch/internal/config"
)

func setupHandler(t *testing.T) (http.Handler, *config.Manager, *Gatekeeper) {
	initialCfg := config.Config{
		Host: "localhost",
		Port: 8080,
	}
	mgr := config.NewManager(initialCfg)
	gate := NewGatekeeper()
	handler := NewHandler(mgr, gate)
	return handler, mgr, gate
}

func TestHandler_Health(t *testing.T) {
	handler, _, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Glitch-Scenario", "test-env")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}

	if res["status"] != "ok" {
		t.Errorf("expected status ok, got %v", res["status"])
	}
	if res["paused"] != false {
		t.Errorf("expected paused false, got %v", res["paused"])
	}
	if res["scenario"] != "test-env" {
		t.Errorf("expected scenario test-env, got %v", res["scenario"])
	}
}

func TestHandler_ConfigAndRules(t *testing.T) {
	handler, mgr, _ := setupHandler(t)
	scenario := "test-rules"

	// 1. Post Rules
	override := config.Config{
		Failure: config.FailureConfig{
			Rate: 100,
			Statuses: []config.StatusConfig{
				{Code: 500, Rate: 100},
			},
		},
	}
	body, _ := json.Marshal(override)
	req := httptest.NewRequest(http.MethodPatch, "/rules", bytes.NewReader(body))
	req.Header.Set("X-Glitch-Scenario", scenario)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	cfg := mgr.Resolve(scenario)
	if cfg.Failure.Rate != 100 {
		t.Errorf("expected failure rate 100, got %v", cfg.Failure.Rate)
	}

	// 2. Get Config
	req2 := httptest.NewRequest(http.MethodGet, "/config", nil)
	req2.Header.Set("X-Glitch-Scenario", scenario)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr2.Code)
	}

	var res config.Config
	if err := json.NewDecoder(rr2.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if res.Failure.Rate != 100 {
		t.Errorf("expected failure rate 100, got %v", res.Failure.Rate)
	}

	// 3. Delete Rules
	req3 := httptest.NewRequest(http.MethodDelete, "/rules", nil)
	req3.Header.Set("X-Glitch-Scenario", scenario)
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr3.Code)
	}

	cfg = mgr.Resolve(scenario)
	if cfg.Failure.Rate != 0 {
		t.Errorf("expected failure rate 0, got %v", cfg.Failure.Rate)
	}
}

func TestHandler_PauseResume(t *testing.T) {
	handler, _, gate := setupHandler(t)
	scenario := "test-pause"

	// Pause
	req := httptest.NewRequest(http.MethodPost, "/pause?timeout=100ms", nil)
	req.Header.Set("X-Glitch-Scenario", scenario)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	if !gate.IsPaused(scenario) {
		t.Fatal("expected gatekeeper to be paused")
	}

	// Resume
	req2 := httptest.NewRequest(http.MethodPost, "/resume", nil)
	req2.Header.Set("X-Glitch-Scenario", scenario)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr2.Code)
	}

	if gate.IsPaused(scenario) {
		t.Fatal("expected gatekeeper to be resumed")
	}
}

func TestHandler_GetBaseline(t *testing.T) {
	handler, _, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/config/baseline", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var res config.Config
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if res.Port != 8080 {
		t.Errorf("expected port 8080, got %d", res.Port)
	}
}

func TestHandler_Profiles(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(filepath.Join(".glitch", "profiles"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".glitch", "profiles", "flaky.yaml"), []byte("name: flaky\nfailure:\n  rate: 25\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	handler, mgr, _ := setupHandler(t)

	// Get Profiles
	req := httptest.NewRequest(http.MethodGet, "/profiles", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var profiles struct {
		Custom []string `json:"custom"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&profiles); err != nil {
		t.Fatalf("failed to decode profiles: %v", err)
	}
	if len(profiles.Custom) != 1 || profiles.Custom[0] != "flaky" {
		t.Errorf("expected custom profile flaky, got %v", profiles.Custom)
	}

	// Post Profile (we'll use a builtin one like "bad-wifi")
	req2 := httptest.NewRequest(http.MethodPost, "/profile/bad-wifi", nil)
	req2.Header.Set("X-Glitch-Scenario", "test-profile")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rr2.Code, rr2.Body.String())
	}

	cfg := mgr.Resolve("test-profile")
	if cfg.Failure.Rate == 0 && cfg.Latency.Fixed.Duration == 0 {
		t.Errorf("expected profile to modify config")
	}

	// Post a discovered custom profile by name.
	reqCustom := httptest.NewRequest(http.MethodPost, "/profile/flaky", nil)
	reqCustom.Header.Set("X-Glitch-Scenario", "custom-profile")
	rrCustom := httptest.NewRecorder()
	handler.ServeHTTP(rrCustom, reqCustom)
	if rrCustom.Code != http.StatusOK {
		t.Fatalf("expected 200 for custom profile, got %d. Body: %s", rrCustom.Code, rrCustom.Body.String())
	}
	if cfg := mgr.Resolve("custom-profile"); cfg.Failure.Rate != 25 {
		t.Errorf("expected custom profile failure rate 25, got %v", cfg.Failure.Rate)
	}

	// Post Profile Invalid
	req3 := httptest.NewRequest(http.MethodPost, "/profile/nonexistent", nil)
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr3.Code)
	}
}

func TestHandler_ScenarioIsolation(t *testing.T) {
	handler, mgr, _ := setupHandler(t)

	// Scenario A creates a rule
	overrideA := config.Config{
		Failure: config.FailureConfig{
			Rate: 100,
			Statuses: []config.StatusConfig{
				{Code: 400, Rate: 100},
			},
		},
	}
	bodyA, _ := json.Marshal(overrideA)
	reqA := httptest.NewRequest(http.MethodPatch, "/rules", bytes.NewReader(bodyA))
	reqA.Header.Set("X-Glitch-Scenario", "scenario-a")
	rrA := httptest.NewRecorder()
	handler.ServeHTTP(rrA, reqA)

	if rrA.Code != http.StatusOK {
		t.Fatalf("scenario-a expected 200, got %d", rrA.Code)
	}

	// Scenario B creates a different rule
	overrideB := config.Config{
		Failure: config.FailureConfig{
			Rate: 50,
			Statuses: []config.StatusConfig{
				{Code: 500, Rate: 50},
			},
		},
	}
	bodyB, _ := json.Marshal(overrideB)
	reqB := httptest.NewRequest(http.MethodPatch, "/rules", bytes.NewReader(bodyB))
	reqB.Header.Set("X-Glitch-Scenario", "scenario-b")
	rrB := httptest.NewRecorder()
	handler.ServeHTTP(rrB, reqB)

	if rrB.Code != http.StatusOK {
		t.Fatalf("scenario-b expected 200, got %d", rrB.Code)
	}

	// Verify Scenario A configuration
	cfgA := mgr.Resolve("scenario-a")
	if cfgA.Failure.Rate != 100 || len(cfgA.Failure.Statuses) == 0 || cfgA.Failure.Statuses[0].Code != 400 {
		t.Errorf("scenario-a isolation failed, got rate %v, status %v", cfgA.Failure.Rate, cfgA.Failure.Statuses)
	}

	// Verify Scenario B configuration
	cfgB := mgr.Resolve("scenario-b")
	if cfgB.Failure.Rate != 50 || len(cfgB.Failure.Statuses) == 0 || cfgB.Failure.Statuses[0].Code != 500 {
		t.Errorf("scenario-b isolation failed, got rate %v, status %v", cfgB.Failure.Rate, cfgB.Failure.Statuses)
	}

	// Verify Baseline configuration is untouched
	cfgBaseline := mgr.Baseline()
	if cfgBaseline.Failure.Rate != 0 || len(cfgBaseline.Failure.Statuses) != 0 {
		t.Errorf("baseline isolation failed, got rate %v, statuses %v", cfgBaseline.Failure.Rate, cfgBaseline.Failure.Statuses)
	}
}
