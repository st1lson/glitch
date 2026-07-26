package control

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/st1lson/glitch/internal/config"
)

// Handler provides the HTTP control API for Glitch.
type Handler struct {
	cfg  *config.Manager
	gate *Gatekeeper
}

// NewHandler creates a new control API handler.
func NewHandler(cfg *config.Manager, gate *Gatekeeper) http.Handler {
	h := &Handler{
		cfg:  cfg,
		gate: gate,
	}

	r := chi.NewRouter()

	r.Get("/health", h.handleHealth)
	
	r.Get("/config", h.handleGetConfig)
	r.Get("/config/baseline", h.handleGetBaseline)
	
	r.Post("/rules", h.handlePostRules)
	r.Delete("/rules", h.handleDeleteRules)
	
	r.Get("/profiles", h.handleGetProfiles)
	r.Post("/profile/{name}", h.handlePostProfile)
	
	r.Post("/pause", h.handlePause)
	r.Post("/resume", h.handleResume)

	return r
}

func extractScenario(r *http.Request) string {
	return r.Header.Get("X-Glitch-Scenario")
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func respondError(w http.ResponseWriter, status int, err string, extras ...map[string]interface{}) {
	resp := map[string]interface{}{"error": err}
	if len(extras) > 0 {
		for k, v := range extras[0] {
			resp[k] = v
		}
	}
	respondJSON(w, status, resp)
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	scenario := extractScenario(r)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"paused":   h.gate.IsPaused(scenario),
		"scenario": scenario,
	})
}

func (h *Handler) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	scenario := extractScenario(r)
	respondJSON(w, http.StatusOK, h.cfg.Resolve(scenario))
}

func (h *Handler) handleGetBaseline(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, h.cfg.Baseline())
}

func (h *Handler) handlePostRules(w http.ResponseWriter, r *http.Request) {
	var override config.Config
	if err := json.NewDecoder(r.Body).Decode(&override); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}

	scenario := extractScenario(r)
	h.cfg.Overlay(scenario, &override)
	respondJSON(w, http.StatusOK, h.cfg.Resolve(scenario))
}

func (h *Handler) handleDeleteRules(w http.ResponseWriter, r *http.Request) {
	scenario := extractScenario(r)
	h.cfg.Reset(scenario)
	respondJSON(w, http.StatusOK, map[string]string{"message": "reset to baseline"})
}

func (h *Handler) handleGetProfiles(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, this would also scan the filesystem for custom profiles.
	// For this version, we just return the built-ins.
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"builtin": config.BuiltinProfileNames(),
		"custom":  []string{},
	})
}

func (h *Handler) handlePostProfile(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	
	profile, err := config.LoadProfile(name)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error(), map[string]interface{}{
			"available": config.BuiltinProfileNames(),
		})
		return
	}

	scenario := extractScenario(r)
	h.cfg.ResetAndApplyProfile(scenario, profile)
	respondJSON(w, http.StatusOK, map[string]string{"message": "profile applied: " + name})
}

func (h *Handler) handlePause(w http.ResponseWriter, r *http.Request) {
	scenario := extractScenario(r)
	var timeout time.Duration
	if tStr := r.URL.Query().Get("timeout"); tStr != "" {
		if d, err := time.ParseDuration(tStr); err == nil {
			timeout = d
		} else {
			respondError(w, http.StatusBadRequest, "invalid timeout format")
			return
		}
	}
	
	h.gate.Pause(scenario, timeout)
	respondJSON(w, http.StatusOK, map[string]string{"message": "paused"})
}

func (h *Handler) handleResume(w http.ResponseWriter, r *http.Request) {
	scenario := extractScenario(r)
	h.gate.Resume(scenario)
	respondJSON(w, http.StatusOK, map[string]string{"message": "resumed"})
}
