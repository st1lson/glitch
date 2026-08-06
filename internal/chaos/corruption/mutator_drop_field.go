package corruption

import (
	"context"

	"github.com/st1lson/glitch/internal/chaos/rng"
)

// FieldDropper removes a random key from an object or element from an array.
type FieldDropper struct{}

func (m *FieldDropper) Name() string { return "drop_field" }

func (m *FieldDropper) Mutate(ctx context.Context, data any) any {
	switch v := data.(type) {
	case map[string]any:
		if len(v) == 0 {
			return v
		}
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		delete(v, keys[rng.FromContext(ctx).IntN(len(keys))])
		return v
	case []any:
		if len(v) == 0 {
			return v
		}
		idx := rng.FromContext(ctx).IntN(len(v))
		return append(v[:idx], v[idx+1:]...)
	default:
		return data
	}
}
