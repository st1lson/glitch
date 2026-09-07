package chaos

import (
	"sync"
	"testing"
)

func TestOperationIndex_RegisterAndLookup(t *testing.T) {
	idx := NewOperationIndex()

	idx.Register("getUser", "GET", "/users/{id}")
	idx.Register("createOrder", "POST", "/orders")

	if idx.Len() != 2 {
		t.Errorf("expected 2 entries, got %d", idx.Len())
	}

	entry, ok := idx.Lookup("getUser")
	if !ok {
		t.Fatal("expected getUser to be found")
	}
	if entry.Method != "GET" || entry.Path != "/users/{id}" {
		t.Errorf("unexpected entry for getUser: %+v", entry)
	}

	entry2, ok := idx.Lookup("createOrder")
	if !ok {
		t.Fatal("expected createOrder to be found")
	}
	if entry2.Method != "POST" || entry2.Path != "/orders" {
		t.Errorf("unexpected entry for createOrder: %+v", entry2)
	}

	_, ok = idx.Lookup("unknown")
	if ok {
		t.Error("expected unknown operation ID to return false")
	}
}

func TestOperationIndex_NilSafe(t *testing.T) {
	var idx *OperationIndex

	idx.Register("foo", "GET", "/foo")
	if idx.Len() != 0 {
		t.Errorf("expected 0 len on nil index")
	}

	entry, ok := idx.Lookup("foo")
	if ok || entry != (OpEntry{}) {
		t.Errorf("expected not found on nil index")
	}
}

func TestOperationIndex_Concurrent(t *testing.T) {
	idx := NewOperationIndex()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			idx.Register("op", "GET", "/path")
			_, _ = idx.Lookup("op")
			_ = idx.Len()
		}(i)
	}

	wg.Wait()
}
