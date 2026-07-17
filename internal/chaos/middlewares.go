package chaos

import (
	"net/http"

	"github.com/st1lson/glitch/internal/chaos/corruption"
	"github.com/st1lson/glitch/internal/chaos/failure"
	"github.com/st1lson/glitch/internal/chaos/latency"
	"github.com/st1lson/glitch/internal/chaos/stall"
	"github.com/st1lson/glitch/internal/chaos/throttle"
	"github.com/st1lson/glitch/internal/config"
)

// BandwidthMiddleware injects bandwidth throttling.
func BandwidthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			eff, ok := getEffectiveChaos(r.Context())
			if ok && eff.Bandwidth != "" {
				if bps, err := config.ParseBandwidth(eff.Bandwidth); err == nil && bps > 0 {
					w = throttle.NewWriter(w, bps)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// LatencyMiddleware injects artificial latency.
func LatencyMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			eff, ok := getEffectiveChaos(r.Context())
			if ok && eff.Latency.Enabled() {
				duration := latency.Inject(r.Context(), eff.Latency)
				if info := GetChaosInfo(r); info != nil {
					info.LatencyAdded = duration
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// FailureMiddleware injects HTTP failures (short-circuits request if triggered).
func FailureMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			eff, ok := getEffectiveChaos(r.Context())
			if ok && eff.Failure.Enabled() {
				if fail, code := failure.ShouldTrigger(eff.Failure); fail {
					if info := GetChaosInfo(r); info != nil {
						info.FailureCode = code
					}
					http.Error(w, http.StatusText(code), code)
					return // short circuit
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// StallMiddleware injects stalls in the response stream.
func StallMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			eff, ok := getEffectiveChaos(r.Context())
			if ok && eff.Stall.Enabled() && stall.ShouldTrigger(eff.Stall) {
				w = stall.NewWriter(w, eff.Stall.Mode, eff.Stall.DropAt)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CorruptionMiddleware injects payload corruption.
func CorruptionMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			eff, ok := getEffectiveChaos(r.Context())
			if ok && eff.Corruption.Enabled() && corruption.ShouldTrigger(eff.Corruption) {
				cw := corruption.NewWriter(w, eff.Corruption)
				if info := GetChaosInfo(r); info != nil {
					info.Corrupted = true
				}
				
				next.ServeHTTP(cw, r)
				cw.Flush()
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
