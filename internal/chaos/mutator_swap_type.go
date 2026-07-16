package chaos

import (
	"math/rand/v2"
)

// TypeSwapper changes a value's type.
type TypeSwapper struct{}

func (m *TypeSwapper) Name() string { return "swap_type" }

func (m *TypeSwapper) Mutate(data any) any {
	// If given a map or slice, pick a random element and swap it
	switch v := data.(type) {
	case map[string]any:
		if len(v) == 0 {
			return v
		}
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		k := keys[rand.IntN(len(keys))]
		v[k] = swapPrimitive(v[k])
		return v
	case []any:
		if len(v) == 0 {
			return v
		}
		idx := rand.IntN(len(v))
		v[idx] = swapPrimitive(v[idx])
		return v
	default:
		return swapPrimitive(data)
	}
}

func swapPrimitive(val any) any {
	switch val.(type) {
	case string:
		return 42 // string -> int
	case float64, int:
		return "corrupted_string" // number -> string
	case bool:
		return 1 // bool -> number
	case nil:
		return "not_null"
	default:
		return 999
	}
}
