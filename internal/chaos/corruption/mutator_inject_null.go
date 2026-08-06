package corruption

import (
	"context"

	"github.com/st1lson/glitch/internal/chaos/rng"
)

// NullInjector replaces a random value with null.
type NullInjector struct{}

func (m *NullInjector) Name() string { return "inject_null" }

func (m *NullInjector) Mutate(ctx context.Context, data any) any {
	switch v := data.(type) {
	case map[string]any:
		if len(v) == 0 {
			return v
		}
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		v[keys[rng.FromContext(ctx).IntN(len(keys))]] = nil
		return v
	case []any:
		if len(v) == 0 {
			return v
		}
		idx := rng.FromContext(ctx).IntN(len(v))
		v[idx] = nil
		return v
	default:
		return nil
	}
}
