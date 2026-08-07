package corruption

import (
	"context"

	"github.com/st1lson/glitch/internal/chaos/rng"
)

// TypeSwapper changes a value's type.
type TypeSwapper struct{}

func (m *TypeSwapper) Name() string { return "swap_type" }

func (m *TypeSwapper) Mutate(ctx context.Context, data any) any {

	switch v := data.(type) {
	case map[string]any:
		if len(v) == 0 {
			return v
		}
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		k := keys[rng.FromContext(ctx).IntN(len(keys))]
		v[k] = swapPrimitive(v[k])
		return v
	case []any:
		if len(v) == 0 {
			return v
		}
		idx := rng.FromContext(ctx).IntN(len(v))
		v[idx] = swapPrimitive(v[idx])
		return v
	default:
		return swapPrimitive(data)
	}
}

func swapPrimitive(val any) any {
	switch val.(type) {
	case string:
		return 42
	case float64, int:
		return "corrupted_string"
	case bool:
		return 1
	case nil:
		return "not_null"
	default:
		return 999
	}
}
