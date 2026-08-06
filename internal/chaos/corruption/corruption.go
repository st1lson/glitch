package corruption

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/st1lson/glitch/internal/chaos/rng"
	"github.com/st1lson/glitch/internal/config"
)

// ShouldTrigger determines whether to inject a payload corruption for the current request.
func ShouldTrigger(ctx context.Context, cfg config.CorruptionConfig) bool {
	if cfg.Rate <= 0 {
		return false
	}
	return rng.FromContext(ctx).Float64() < (cfg.Rate / 100.0)
}

// Writer wraps an http.ResponseWriter to buffer and mutate the response body.
type Writer struct {
	http.ResponseWriter
	buf         bytes.Buffer
	cfg         config.CorruptionConfig
	statusCode  int
	wroteHeader bool
	ctx         context.Context
}

// NewWriter creates a new Writer.
func NewWriter(w http.ResponseWriter, cfg config.CorruptionConfig, ctx context.Context) *Writer {
	return &Writer{
		ResponseWriter: w,
		cfg:            cfg,
		ctx:            ctx,
	}
}

// WriteHeader captures the status code but delays writing it until flush.
func (c *Writer) WriteHeader(statusCode int) {
	if !c.wroteHeader {
		c.statusCode = statusCode
		c.wroteHeader = true
	}
}

// Write buffers the response body.
func (c *Writer) Write(p []byte) (int, error) {
	if !c.wroteHeader {
		c.WriteHeader(http.StatusOK)
	}
	return c.buf.Write(p)
}

// Flush applies corruption if applicable and writes the buffered data to the underlying writer.
func (c *Writer) Flush() {
	if !c.wroteHeader {
		return
	}

	contentType := c.Header().Get("Content-Type")
	body := c.buf.Bytes()

	// Only corrupt JSON responses.
	if strings.Contains(contentType, "application/json") && len(body) > 0 {
		corruptedBody, _ := CorruptPayload(c.ctx, body, c.cfg)
		body = corruptedBody
	}

	c.Header().Set("Content-Length", strconv.Itoa(len(body)))

	c.ResponseWriter.WriteHeader(c.statusCode)
	c.ResponseWriter.Write(body)
}

// Mutator defines a strategy for mutating JSON data.
type Mutator interface {
	Name() string
	Mutate(ctx context.Context, data any) any
}

// CorruptPayload applies random mutation strategies to the JSON payload.
func CorruptPayload(ctx context.Context, body []byte, cfg config.CorruptionConfig) ([]byte, string) {
	mutators := getMutators(cfg.Strategies)
	if len(mutators) == 0 {
		return body, "" // No valid mutators
	}

	numMutators := 1
	if cfg.Multi {
		numMutators = rng.FromContext(ctx).IntN(3) + 2 // 2 to 4 mutators if multi is enabled
	}

	mutatedNames := []string{}
	currentBody := body

	for i := 0; i < numMutators; i++ {
		mutator := mutators[rng.FromContext(ctx).IntN(len(mutators))]
		mutatedNames = append(mutatedNames, mutator.Name())

		if mutator.Name() == "break_syntax" {
			currentBody = mutator.Mutate(ctx, currentBody).([]byte)
		} else {
			var data any
			if err := json.Unmarshal(currentBody, &data); err != nil {
				// If we can't unmarshal (e.g. invalid JSON from a previous break_syntax), stop and return
				break
			}
			data = walkAndMutate(ctx, data, mutator, rng.FromContext(ctx).IntN(3)+1) // Walk 1 to 3 levels
			newBody, err := json.Marshal(data)
			if err == nil {
				currentBody = newBody
			}
		}
	}

	return currentBody, strings.Join(mutatedNames, ",")
}

func walkAndMutate(ctx context.Context, data any, mutator Mutator, depthLeft int) any {
	if depthLeft <= 0 {
		return mutator.Mutate(ctx, data)
	}

	switch v := data.(type) {
	case map[string]any:
		if len(v) == 0 {
			return mutator.Mutate(ctx, data)
		}
		// Pick a random key to recurse into or mutate directly if it's the target depth
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		key := keys[rng.FromContext(ctx).IntN(len(keys))]

		if depthLeft == 1 {
			// At target depth, pass the whole map to let mutator pick what to do (e.g. drop a key)
			// Wait, the mutator interface says `Mutate(data any) any`.
			// It's cleaner if the mutator receives the container (map/slice) and mutates it.
			return mutator.Mutate(ctx, data)
		} else {
			v[key] = walkAndMutate(ctx, v[key], mutator, depthLeft-1)
			return v
		}

	case []any:
		if len(v) == 0 {
			return mutator.Mutate(ctx, data)
		}
		if depthLeft == 1 {
			return mutator.Mutate(ctx, data)
		}
		idx := rng.FromContext(ctx).IntN(len(v))
		v[idx] = walkAndMutate(ctx, v[idx], mutator, depthLeft-1)
		return v

	default:
		// Primitive value, just mutate it directly
		return mutator.Mutate(ctx, data)
	}
}

// Built-in mutators

func getMutators(strategies []config.CorruptionStrategy) []Mutator {
	allMutators := map[config.CorruptionStrategy]Mutator{
		config.StrategyDropField:   &FieldDropper{},
		config.StrategySwapType:    &TypeSwapper{},
		config.StrategyInjectNull:  &NullInjector{},
		config.StrategyBreakSyntax: &SyntaxBreaker{},
	}

	var active []Mutator
	for _, s := range strategies {
		if m, ok := allMutators[s]; ok {
			active = append(active, m)
		}
	}
	return active
}
