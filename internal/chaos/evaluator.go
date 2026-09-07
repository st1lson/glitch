package chaos

import (
	"context"
	"net/http"

	"github.com/st1lson/glitch/internal/config"
)

// effectiveChaosKey is used to store the evaluated EffectiveChaos in the request context.
type effectiveChaosKey struct{}

// getEffectiveChaos retrieves the EffectiveChaos from the request context.
func getEffectiveChaos(ctx context.Context) (EffectiveChaos, bool) {
	eff, ok := ctx.Value(effectiveChaosKey{}).(EffectiveChaos)
	return eff, ok
}

// setEffectiveChaos stores the EffectiveChaos in the request context.
func setEffectiveChaos(ctx context.Context, eff EffectiveChaos) context.Context {
	return context.WithValue(ctx, effectiveChaosKey{}, eff)
}

// EffectiveChaos represents the final chaos settings for a specific request.
type EffectiveChaos struct {
	Bandwidth  config.Bandwidth
	Latency    config.LatencyConfig
	Failure    config.FailureConfig
	Stall      config.StallConfig
	Corruption config.CorruptionConfig
	Realtime   config.RealtimeConfig
}

// evalChaos overlays route-specific chaos on top of global chaos, selecting the most specific match.
func evalChaos(cfg config.Config, r *http.Request, opIdx *OperationIndex) EffectiveChaos {
	eff := EffectiveChaos{
		Bandwidth:  cfg.Bandwidth,
		Latency:    cfg.Latency,
		Failure:    cfg.Failure,
		Stall:      cfg.Stall,
		Corruption: cfg.Corruption,
		Realtime:   cfg.Realtime,
	}

	if len(cfg.Routes) == 0 {
		return eff
	}

	var bestMatch *config.RouteConfig
	bestScore := -1

	for i := range cfg.Routes {
		route := &cfg.Routes[i]
		predicates := BuildPredicates(route, opIdx)
		if len(predicates) == 0 {
			continue
		}

		allMatched := true
		score := 0
		for _, p := range predicates {
			matched, s := p.Match(r)
			if !matched {
				allMatched = false
				break
			}
			score += s
		}

		if allMatched && score > bestScore {
			bestScore = score
			bestMatch = route
		}
	}

	if bestMatch != nil {
		if bestMatch.Bandwidth != nil {
			eff.Bandwidth = *bestMatch.Bandwidth
		}
		if bestMatch.Latency != nil {
			eff.Latency = *bestMatch.Latency
		}
		if bestMatch.Failure != nil {
			eff.Failure = *bestMatch.Failure
		}
		if bestMatch.Stall != nil {
			eff.Stall = *bestMatch.Stall
		}
		if bestMatch.Corruption != nil {
			eff.Corruption = *bestMatch.Corruption
		}
		if bestMatch.Realtime != nil {
			eff.Realtime = *bestMatch.Realtime
		}
	}

	return eff
}
