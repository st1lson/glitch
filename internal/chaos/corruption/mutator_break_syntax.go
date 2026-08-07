package corruption

import (
	"context"
	"strings"

	"github.com/st1lson/glitch/internal/chaos/rng"
)

// SyntaxBreaker corrupts the raw JSON string.
type SyntaxBreaker struct{}

func (m *SyntaxBreaker) Name() string { return "break_syntax" }

func (m *SyntaxBreaker) Mutate(ctx context.Context, data any) any {
	b, ok := data.([]byte)
	if !ok {
		return data
	}
	if len(b) < 2 {
		return b
	}

	choice := rng.FromContext(ctx).IntN(3)
	switch choice {
	case 0:

		return b[:len(b)/2]
	case 1:

		str := string(b)
		if strings.HasSuffix(strings.TrimSpace(str), "}") {
			str = strings.TrimSpace(str)
			return []byte(str[:len(str)-1] + ",}")
		} else if strings.HasSuffix(strings.TrimSpace(str), "]") {
			str = strings.TrimSpace(str)
			return []byte(str[:len(str)-1] + ",]")
		}
		return append(b, ']')
	case 2:

		idx := len(b) / 2
		res := make([]byte, 0, len(b)+1)
		res = append(res, b[:idx]...)
		res = append(res, '"')
		res = append(res, b[idx:]...)
		return res
	}
	return b
}
