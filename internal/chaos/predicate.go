package chaos

import (
	"net/http"
	"strings"

	"github.com/st1lson/glitch/internal/config"
)

type Predicate interface {
	Match(r *http.Request) (matched bool, score int)
}

type PathPredicate struct {
	Pattern string
}

func (p *PathPredicate) Match(r *http.Request) (bool, int) {
	return matchPath(p.Pattern, r.URL.Path)
}

type MethodPredicate struct {
	Method string
}

func (p *MethodPredicate) Match(r *http.Request) (bool, int) {
	if strings.EqualFold(p.Method, r.Method) {
		return true, 100
	}
	return false, 0
}

type OperationIDPredicate struct {
	OperationID string
	Index       *OperationIndex
}

func (p *OperationIDPredicate) Match(r *http.Request) (bool, int) {
	if p.Index == nil {
		return false, 0
	}

	entry, ok := p.Index.Lookup(p.OperationID)
	if !ok {
		return false, 0
	}

	if entry.Method != "" && !strings.EqualFold(entry.Method, r.Method) {
		return false, 0
	}

	if !matchOpenAPIPath(entry.Path, r.URL.Path) {
		return false, 0
	}

	return true, 2000
}

type HeadersPredicate struct {
	Headers map[string]string
}

func (p *HeadersPredicate) Match(r *http.Request) (bool, int) {
	if len(p.Headers) == 0 {
		return true, 0
	}

	for k, v := range p.Headers {
		got := r.Header.Get(k)
		if got == "" || !strings.EqualFold(got, v) {
			return false, 0
		}
	}

	return true, 50 * len(p.Headers)
}

type QueryPredicate struct {
	Query map[string]string
}

func (p *QueryPredicate) Match(r *http.Request) (bool, int) {
	if len(p.Query) == 0 {
		return true, 0
	}

	queryParams := r.URL.Query()
	for k, v := range p.Query {
		got := queryParams.Get(k)
		if got == "" || !strings.EqualFold(got, v) {
			return false, 0
		}
	}

	return true, 25 * len(p.Query)
}

type BodyPredicateMatcher struct {
	Conditions []config.BodyPredicate
}

func (p *BodyPredicateMatcher) Match(r *http.Request) (bool, int) {
	if len(p.Conditions) == 0 {
		return true, 0
	}

	data, _ := PeekJSONBody(r)
	for _, cond := range p.Conditions {
		val, exists := GetNestedValue(data, cond.Field)
		if !EvaluateBodyCondition(val, exists, cond.Op, cond.Value) {
			return false, 0
		}
	}

	return true, 200 * len(p.Conditions)
}

func BuildPredicates(route *config.RouteConfig, opIdx *OperationIndex) []Predicate {
	if route == nil {
		return nil
	}

	var predicates []Predicate

	if route.OperationID != "" {
		predicates = append(predicates, &OperationIDPredicate{
			OperationID: route.OperationID,
			Index:       opIdx,
		})
	}

	if route.Path != "" {
		predicates = append(predicates, &PathPredicate{
			Pattern: route.Path,
		})
	}

	if route.Method != "" {
		predicates = append(predicates, &MethodPredicate{
			Method: route.Method,
		})
	}

	if len(route.Headers) > 0 {
		predicates = append(predicates, &HeadersPredicate{
			Headers: route.Headers,
		})
	}

	if len(route.Query) > 0 {
		predicates = append(predicates, &QueryPredicate{
			Query: route.Query,
		})
	}

	if len(route.Body) > 0 {
		predicates = append(predicates, &BodyPredicateMatcher{
			Conditions: route.Body,
		})
	}

	return predicates
}

func matchPath(pattern, path string) (bool, int) {
	if pattern == path {
		return true, 1000 + len(pattern)
	}
	if before, ok := strings.CutSuffix(pattern, "*"); ok {
		prefix := before
		if strings.HasPrefix(path, prefix) {
			return true, len(prefix)
		}
	}
	return false, 0
}

func matchOpenAPIPath(template, path string) bool {
	if template == path || template == "*" {
		return true
	}
	if before, ok := strings.CutSuffix(template, "*"); ok {
		return strings.HasPrefix(path, before)
	}

	tTrimmed := strings.Trim(template, "/")
	pTrimmed := strings.Trim(path, "/")

	if tTrimmed == "" && pTrimmed == "" {
		return true
	}
	if tTrimmed == "" || pTrimmed == "" {
		return false
	}

	tParts := strings.Split(tTrimmed, "/")
	pParts := strings.Split(pTrimmed, "/")

	if len(tParts) != len(pParts) {
		return false
	}

	for i := range tParts {
		tSeg := tParts[i]
		pSeg := pParts[i]

		if strings.HasPrefix(tSeg, "{") && strings.HasSuffix(tSeg, "}") {
			if pSeg == "" {
				return false
			}
			continue
		}

		if !strings.EqualFold(tSeg, pSeg) {
			return false
		}
	}

	return true
}
