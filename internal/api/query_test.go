package api

import (
	"net/url"
	"testing"
)

func TestApplyQuery_Filtering(t *testing.T) {
	items := []map[string]any{
		{"id": 1, "name": "Apple", "inStock": true, "category": "fruit"},
		{"id": 2, "name": "Banana", "inStock": true, "category": "fruit"},
		{"id": 3, "name": "Carrot", "inStock": false, "category": "vegetable"},
	}

	q := url.Values{}
	q.Add("category", "fruit")
	q.Add("inStock", "true")

	result, total := ApplyQuery(items, q)

	if total != 2 {
		t.Errorf("expected 2 items, got %d", total)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 items in result, got %d", len(result))
	}
}

func TestApplyQuery_Sorting(t *testing.T) {
	items := []map[string]any{
		{"id": 1, "price": 10.5},
		{"id": 2, "price": 5.0},
		{"id": 3, "price": 15.0},
	}

	q := url.Values{}
	q.Add("_sort", "price")
	q.Add("_order", "asc")

	result, _ := ApplyQuery(items, q)

	if result[0]["id"] != 2 {
		t.Errorf("expected id 2 to be first, got %v", result[0]["id"])
	}
	if result[2]["id"] != 3 {
		t.Errorf("expected id 3 to be last, got %v", result[2]["id"])
	}

	q.Set("_order", "desc")
	resultDesc, _ := ApplyQuery(items, q)

	if resultDesc[0]["id"] != 3 {
		t.Errorf("expected id 3 to be first in desc, got %v", resultDesc[0]["id"])
	}
}

func TestApplyQuery_Pagination(t *testing.T) {
	items := []map[string]any{
		{"id": 1}, {"id": 2}, {"id": 3}, {"id": 4}, {"id": 5},
	}

	q := url.Values{}
	q.Add("_page", "2")
	q.Add("_limit", "2")

	result, total := ApplyQuery(items, q)

	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 items on page 2, got %d", len(result))
	}
	if result[0]["id"] != 3 {
		t.Errorf("expected first item to be id 3, got %v", result[0]["id"])
	}
}
