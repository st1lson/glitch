package corruption

import (
	"math/rand/v2"
	"strings"
)

// SyntaxBreaker corrupts the raw JSON string.
type SyntaxBreaker struct{}

func (m *SyntaxBreaker) Name() string { return "break_syntax" }

func (m *SyntaxBreaker) Mutate(data any) any {
	b, ok := data.([]byte)
	if !ok {
		return data
	}
	if len(b) < 2 {
		return b
	}

	choice := rand.IntN(3)
	switch choice {
	case 0:
		// Truncate
		return b[:len(b)/2]
	case 1:
		// Inject a trailing comma at the end before closing brace/bracket if possible
		str := string(b)
		if strings.HasSuffix(strings.TrimSpace(str), "}") {
			str = strings.TrimSpace(str)
			return []byte(str[:len(str)-1] + ",}")
		} else if strings.HasSuffix(strings.TrimSpace(str), "]") {
			str = strings.TrimSpace(str)
			return []byte(str[:len(str)-1] + ",]")
		}
		return append(b, ']') // just append junk
	case 2:
		// Inject unescaped quote
		idx := len(b) / 2
		res := make([]byte, 0, len(b)+1)
		res = append(res, b[:idx]...)
		res = append(res, '"')
		res = append(res, b[idx:]...)
		return res
	}
	return b
}
