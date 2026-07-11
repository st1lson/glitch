package api

import (
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// ApplyQuery applies filtering, sorting, and pagination to a list of items
// based on URL query parameters. It returns the resulting items and the total
// count of items after filtering (before pagination), useful for X-Total-Count.
//
// Processing order: filter → sort → paginate.
func ApplyQuery(items []map[string]any, params url.Values) ([]map[string]any, int) {
	result := applyFilters(items, params)
	result = applySort(result, params)

	total := len(result)

	result = applyPagination(result, params)

	// Never return nil — always return an empty slice for clean JSON.
	if result == nil {
		result = []map[string]any{}
	}

	return result, total
}

// applyFilters keeps only items matching all non-underscore query parameters.
// Multiple filters are AND-ed. Comparison is case-insensitive for strings
// and numeric if both values parse as numbers.
func applyFilters(items []map[string]any, params url.Values) []map[string]any {
	filters := make(map[string]string)
	for key, vals := range params {
		if strings.HasPrefix(key, "_") || len(vals) == 0 {
			continue
		}
		filters[key] = vals[0]
	}

	if len(filters) == 0 {
		// Return a copy so upstream mutations don't affect the original.
		out := make([]map[string]any, len(items))
		copy(out, items)
		return out
	}

	var result []map[string]any
	for _, item := range items {
		if matchesAllFilters(item, filters) {
			result = append(result, item)
		}
	}
	return result
}

// matchesAllFilters returns true if the item matches every filter.
func matchesAllFilters(item map[string]any, filters map[string]string) bool {
	for field, want := range filters {
		val, ok := item[field]
		if !ok {
			return false
		}
		if !valuesMatch(val, want) {
			return false
		}
	}
	return true
}

// valuesMatch compares an item field value against the query parameter string.
// It tries numeric comparison first, then falls back to case-insensitive strings.
func valuesMatch(fieldVal any, queryVal string) bool {
	// Try numeric comparison.
	if fieldNum, ok := toFloat64(fieldVal); ok {
		if queryNum, err := strconv.ParseFloat(queryVal, 64); err == nil {
			return fieldNum == queryNum
		}
	}

	// Fall back to case-insensitive string comparison.
	return strings.EqualFold(fmt.Sprintf("%v", fieldVal), queryVal)
}

// toFloat64 attempts to convert an interface value to float64.
func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json_number:
		f, err := n.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(n, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

// json_number is a type alias for json.Number used in type assertions.
// We use a named type to avoid importing encoding/json just for this.
type json_number = interface{ Float64() (float64, error) }

// applySort sorts items by the field specified in _sort, in the direction
// specified by _order (asc or desc, default asc).
func applySort(items []map[string]any, params url.Values) []map[string]any {
	sortField := params.Get("_sort")
	if sortField == "" {
		return items
	}

	order := strings.ToLower(params.Get("_order"))
	descending := order == "desc"

	sort.SliceStable(items, func(i, j int) bool {
		a := items[i][sortField]
		b := items[j][sortField]
		cmp := compareValues(a, b)
		if descending {
			return cmp > 0
		}
		return cmp < 0
	})

	return items
}

// compareValues compares two interface values for sorting.
// Returns -1, 0, or 1 like a three-way comparison.
func compareValues(a, b any) int {
	// Handle nil values — push them to the end.
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return 1
	}
	if b == nil {
		return -1
	}

	// Try numeric comparison.
	aNum, aOk := toFloat64(a)
	bNum, bOk := toFloat64(b)
	if aOk && bOk {
		switch {
		case aNum < bNum:
			return -1
		case aNum > bNum:
			return 1
		default:
			return 0
		}
	}

	// Fall back to string comparison.
	aStr := strings.ToLower(fmt.Sprintf("%v", a))
	bStr := strings.ToLower(fmt.Sprintf("%v", b))
	switch {
	case aStr < bStr:
		return -1
	case aStr > bStr:
		return 1
	default:
		return 0
	}
}

// applyPagination returns a slice of items for the requested page.
// _page is 1-based, _limit defaults to 10.
func applyPagination(items []map[string]any, params url.Values) []map[string]any {
	pageStr := params.Get("_page")
	if pageStr == "" {
		return items
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit := 10
	if limitStr := params.Get("_limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	start := (page - 1) * limit
	if start >= len(items) {
		return []map[string]any{}
	}

	end := int(math.Min(float64(start+limit), float64(len(items))))
	return items[start:end]
}
