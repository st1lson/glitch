package chaos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type bodyContextKey struct{}

const MaxBodyReadBytes = 1024 * 1024

func PeekJSONBody(r *http.Request) (map[string]any, *http.Request) {
	if r == nil || r.Body == nil || r.Body == http.NoBody {
		return nil, r
	}

	if cached, ok := r.Context().Value(bodyContextKey{}).(map[string]any); ok {
		return cached, r
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, MaxBodyReadBytes))
	if err != nil {
		return nil, r
	}
	_ = r.Body.Close()

	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	if len(bodyBytes) == 0 {
		return nil, r
	}

	var parsed map[string]any
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		return nil, r
	}

	ctx := context.WithValue(r.Context(), bodyContextKey{}, parsed)
	return parsed, r.WithContext(ctx)
}

func GetNestedValue(data map[string]any, path string) (any, bool) {
	if data == nil || path == "" {
		return nil, false
	}

	parts := strings.Split(path, ".")
	var current any = data

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		val, exists := m[part]
		if !exists {
			return nil, false
		}
		current = val
	}

	return current, true
}

func EvaluateBodyCondition(actual any, exists bool, op, expected string) bool {
	op = strings.ToLower(strings.TrimSpace(op))
	if op == "" {
		op = "eq"
	}

	switch op {
	case "exists":
		if strings.EqualFold(expected, "false") {
			return !exists
		}
		return exists

	case "eq":
		if !exists {
			return false
		}
		return valueMatches(actual, expected)

	case "neq":
		if !exists {
			return true
		}
		return !valueMatches(actual, expected)

	case "contains":
		if !exists {
			return false
		}
		return valueContains(actual, expected)

	case "prefix":
		if !exists {
			return false
		}
		return strings.HasPrefix(strings.ToLower(fmt.Sprintf("%v", actual)), strings.ToLower(expected))

	default:
		return exists && valueMatches(actual, expected)
	}
}

func valueMatches(actual any, expected string) bool {
	switch v := actual.(type) {
	case bool:
		if expBool, err := strconv.ParseBool(expected); err == nil {
			return v == expBool
		}
	case float64:
		if expNum, err := strconv.ParseFloat(expected, 64); err == nil {
			return v == expNum
		}
	case int:
		if expNum, err := strconv.Atoi(expected); err == nil {
			return v == expNum
		}
	case int64:
		if expNum, err := strconv.ParseInt(expected, 10, 64); err == nil {
			return v == expNum
		}
	}

	return strings.EqualFold(fmt.Sprintf("%v", actual), expected)
}

func valueContains(actual any, expected string) bool {
	switch v := actual.(type) {
	case string:
		return strings.Contains(strings.ToLower(v), strings.ToLower(expected))
	case []any:
		for _, item := range v {
			if valueMatches(item, expected) {
				return true
			}
		}
	}
	return strings.Contains(strings.ToLower(fmt.Sprintf("%v", actual)), strings.ToLower(expected))
}
