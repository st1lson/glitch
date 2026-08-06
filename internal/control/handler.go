package control

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/st1lson/glitch/internal/config"
	"github.com/st1lson/glitch/internal/constants"
	"github.com/st1lson/glitch/internal/reporting"
)

// Handler provides the HTTP control API for Glitch.
type Handler struct {
	cfg     *config.Manager
	gate    *Gatekeeper
	reports *reporting.ReportManager
}

// NewHandler creates a new control API handler.
func NewHandler(cfg *config.Manager, gate *Gatekeeper, reports *reporting.ReportManager) http.Handler {
	h := &Handler{
		cfg:     cfg,
		gate:    gate,
		reports: reports,
	}

	r := chi.NewRouter()

	r.Use(authMiddleware(cfg))

	r.Get("/health", h.handleHealth)

	r.Get("/config", h.handleGetConfig)
	r.Get("/config/baseline", h.handleGetBaseline)

	r.Patch("/rules", h.handlePostRules) // PATCH is semantically more correct since it merges
	r.Delete("/rules", h.handleDeleteRules)

	r.Get("/profiles", h.handleGetProfiles)
	r.Post("/profile/{name}", h.handlePostProfile)

	r.Post("/pause", h.handlePause)
	r.Post("/resume", h.handleResume)

	r.Get("/report", h.handleGetReport)
	r.Get("/scenarios/{id}/report", h.handleGetScenarioReport)

	return r
}

func authMiddleware(cfg *config.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			globalCfg := cfg.Baseline()

			if globalCfg.InsecureControlAPI {
				next.ServeHTTP(w, r)
				return
			}

			if globalCfg.ControlToken != "" {
				authHeader := r.Header.Get(constants.HeaderAuthorization)
				expectedHeader := constants.BearerPrefix + globalCfg.ControlToken
				if len(authHeader) == len(expectedHeader) && subtle.ConstantTimeCompare([]byte(authHeader), []byte(expectedHeader)) == 1 {
					next.ServeHTTP(w, r)
					return
				}
				respondError(w, http.StatusUnauthorized, "unauthorized control API access")
				return
			}

			// No token configured: allow only loopback
			if isLoopback(r.RemoteAddr) {
				next.ServeHTTP(w, r)
				return
			}

			respondError(w, http.StatusUnauthorized, "unauthorized control API access")
		})
	}
}

func isLoopback(remoteAddr string) bool {
	// RemoteAddr is usually "IP:port"
	host := remoteAddr
	for i := len(remoteAddr) - 1; i >= 0; i-- {
		if remoteAddr[i] == ':' {
			host = remoteAddr[:i]
			break
		}
	}
	
	// Handle IPv6 brackets like "[::1]:port"
	if len(host) > 2 && host[0] == '[' && host[len(host)-1] == ']' {
		host = host[1 : len(host)-1]
	}

	if host == constants.LocalhostName || host == constants.LocalhostIPv4 || host == constants.LocalhostIPv6 {
		return true
	}

	return false
}

func extractScenario(r *http.Request) string {
	return r.Header.Get(constants.HeaderScenario)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
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
	custom, err := config.CustomProfileNames()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"builtin": config.BuiltinProfileNames(),
		"custom":  custom,
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

func (h *Handler) handleGetReport(w http.ResponseWriter, r *http.Request) {
	reports := h.reports.GetAllReports()
	respondJSON(w, http.StatusOK, reports)
}

func (h *Handler) handleGetScenarioReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	report, ok := h.reports.GetReport(id)
	if !ok {
		respondError(w, http.StatusNotFound, "scenario not found")
		return
	}
	respondJSON(w, http.StatusOK, report)
}
