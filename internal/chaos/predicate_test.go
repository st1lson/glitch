package chaos

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/st1lson/glitch/internal/config"
)

func TestPathPredicate(t *testing.T) {
	tests := []struct {
		pattern   string
		reqPath   string
		wantMatch bool
		wantScore int
	}{
		{"/api/checkout", "/api/checkout", true, 1013},
		{"/api/checkout", "/api/products", false, 0},
		{"/api/*", "/api/checkout", true, 5},
		{"*", "/api/checkout", true, 0},
		{"/api/products/*", "/api/products/123", true, 14},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.reqPath, func(t *testing.T) {
			p := &PathPredicate{Pattern: tt.pattern}
			req := httptest.NewRequest(http.MethodGet, tt.reqPath, nil)
			matched, score := p.Match(req)
			if matched != tt.wantMatch {
				t.Errorf("matched = %v, want %v", matched, tt.wantMatch)
			}
			if score != tt.wantScore {
				t.Errorf("score = %v, want %v", score, tt.wantScore)
			}
		})
	}
}

func TestMethodPredicate(t *testing.T) {
	p := &MethodPredicate{Method: "POST"}

	reqPost := httptest.NewRequest(http.MethodPost, "/test", nil)
	matched, score := p.Match(reqPost)
	if !matched || score != 100 {
		t.Errorf("expected POST to match with score 100, got (%v, %d)", matched, score)
	}

	reqGet := httptest.NewRequest(http.MethodGet, "/test", nil)
	matchedGet, scoreGet := p.Match(reqGet)
	if matchedGet || scoreGet != 0 {
		t.Errorf("expected GET not to match POST predicate")
	}
}

func TestOperationIDPredicate(t *testing.T) {
	idx := NewOperationIndex()
	idx.Register("getUser", "GET", "/users/{id}")
	idx.Register("createOrder", "POST", "/orders")

	p := &OperationIDPredicate{OperationID: "getUser", Index: idx}

	reqMatch := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	matched, score := p.Match(reqMatch)
	if !matched || score != 2000 {
		t.Errorf("expected getUser to match /users/123 with score 2000, got (%v, %d)", matched, score)
	}

	reqWrongMethod := httptest.NewRequest(http.MethodPost, "/users/123", nil)
	matchedWM, _ := p.Match(reqWrongMethod)
	if matchedWM {
		t.Error("expected method mismatch not to match")
	}

	reqWrongPath := httptest.NewRequest(http.MethodGet, "/products/123", nil)
	matchedWP, _ := p.Match(reqWrongPath)
	if matchedWP {
		t.Error("expected path mismatch not to match")
	}

	// Missing in index
	pMissing := &OperationIDPredicate{OperationID: "unknown", Index: idx}
	matchedMiss, _ := pMissing.Match(reqMatch)
	if matchedMiss {
		t.Error("expected unknown op ID not to match")
	}

	// Nil index
	pNil := &OperationIDPredicate{OperationID: "getUser", Index: nil}
	matchedNil, _ := pNil.Match(reqMatch)
	if matchedNil {
		t.Error("expected nil index not to match")
	}
}

func TestHeadersPredicate(t *testing.T) {
	p := &HeadersPredicate{
		Headers: map[string]string{
			"X-Priority": "high",
			"X-Env":      "staging",
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Priority", "high")
	req.Header.Set("X-Env", "staging")

	matched, score := p.Match(req)
	if !matched || score != 100 {
		t.Errorf("expected headers match with score 100, got (%v, %d)", matched, score)
	}

	reqMissing := httptest.NewRequest(http.MethodGet, "/", nil)
	reqMissing.Header.Set("X-Priority", "high")
	matchedMiss, _ := p.Match(reqMissing)
	if matchedMiss {
		t.Error("expected missing header not to match")
	}

	reqWrong := httptest.NewRequest(http.MethodGet, "/", nil)
	reqWrong.Header.Set("X-Priority", "low")
	reqWrong.Header.Set("X-Env", "staging")
	matchedWrong, _ := p.Match(reqWrong)
	if matchedWrong {
		t.Error("expected header value mismatch not to match")
	}
}

func TestQueryPredicate(t *testing.T) {
	p := &QueryPredicate{
		Query: map[string]string{
			"category": "books",
			"sort":     "desc",
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/products?category=books&sort=desc", nil)
	matched, score := p.Match(req)
	if !matched || score != 50 {
		t.Errorf("expected query match with score 50, got (%v, %d)", matched, score)
	}

	reqMissing := httptest.NewRequest(http.MethodGet, "/products?category=books", nil)
	matchedMiss, _ := p.Match(reqMissing)
	if matchedMiss {
		t.Error("expected missing query param not to match")
	}

	reqWrong := httptest.NewRequest(http.MethodGet, "/products?category=electronics&sort=desc", nil)
	matchedWrong, _ := p.Match(reqWrong)
	if matchedWrong {
		t.Error("expected query value mismatch not to match")
	}
}

func TestBodyPredicateMatcher(t *testing.T) {
	p := &BodyPredicateMatcher{
		Conditions: []config.BodyPredicate{
			{Field: "user.role", Op: "eq", Value: "admin"},
			{Field: "plan.type", Op: "prefix", Value: "pro"},
		},
	}

	jsonPayload := `{"user": {"role": "admin"}, "plan": {"type": "professional"}}`
	req := httptest.NewRequest(http.MethodPost, "/checkout", bytes.NewBufferString(jsonPayload))
	matched, score := p.Match(req)
	if !matched || score != 400 {
		t.Errorf("expected body match with score 400, got (%v, %d)", matched, score)
	}

	jsonPayloadFail := `{"user": {"role": "user"}, "plan": {"type": "professional"}}`
	reqFail := httptest.NewRequest(http.MethodPost, "/checkout", bytes.NewBufferString(jsonPayloadFail))
	matchedFail, _ := p.Match(reqFail)
	if matchedFail {
		t.Error("expected body mismatch not to match")
	}
}

func TestBuildPredicates(t *testing.T) {
	idx := NewOperationIndex()
	rc := &config.RouteConfig{
		Path:        "/api/test",
		Method:      "POST",
		OperationID: "testOp",
		Headers:     map[string]string{"H": "V"},
		Query:       map[string]string{"Q": "1"},
		Body:        []config.BodyPredicate{{Field: "f", Op: "eq", Value: "v"}},
	}

	predicates := BuildPredicates(rc, idx)
	if len(predicates) != 6 {
		t.Errorf("expected 6 predicates, got %d", len(predicates))
	}

	emptyPreds := BuildPredicates(nil, idx)
	if len(emptyPreds) != 0 {
		t.Errorf("expected 0 predicates for nil route")
	}
}

func TestMatchOpenAPIPath(t *testing.T) {
	tests := []struct {
		template string
		path     string
		want     bool
	}{
		{"/users/{id}", "/users/123", true},
		{"/users/{id}/posts/{postId}", "/users/42/posts/99", true},
		{"/users/{id}", "/users/123/extra", false},
		{"/users/{id}", "/users", false},
		{"/users", "/users", true},
		{"/users/*", "/users/123/extra", true},
		{"*", "/anything", true},
	}

	for _, tt := range tests {
		t.Run(tt.template+"_"+tt.path, func(t *testing.T) {
			got := matchOpenAPIPath(tt.template, tt.path)
			if got != tt.want {
				t.Errorf("matchOpenAPIPath(%q, %q) = %v, want %v", tt.template, tt.path, got, tt.want)
			}
		})
	}
}
