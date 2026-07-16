package chaos

import (
	"math/rand/v2"
)

// NullInjector replaces a random value with null.
type NullInjector struct{}

func (m *NullInjector) Name() string { return "inject_null" }

func (m *NullInjector) Mutate(data any) any {
	switch v := data.(type) {
	case map[string]any:
		if len(v) == 0 {
			return v
		}
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		v[keys[rand.IntN(len(keys))]] = nil
		return v
	case []any:
		if len(v) == 0 {
			return v
		}
		idx := rand.IntN(len(v))
		v[idx] = nil
		return v
	default:
		return nil
	}
}
