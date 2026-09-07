package chaos

import (
	"strings"
	"sync"
)

type OpEntry struct {
	Method string
	Path   string
}

type OperationIndex struct {
	mu      sync.RWMutex
	entries map[string]OpEntry
}

func NewOperationIndex() *OperationIndex {
	return &OperationIndex{
		entries: make(map[string]OpEntry),
	}
}

func (idx *OperationIndex) Register(operationID, method, path string) {
	if idx == nil {
		return
	}
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.entries[operationID] = OpEntry{
		Method: strings.ToUpper(strings.TrimSpace(method)),
		Path:   strings.TrimSpace(path),
	}
}

func (idx *OperationIndex) Lookup(operationID string) (OpEntry, bool) {
	if idx == nil {
		return OpEntry{}, false
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	entry, ok := idx.entries[operationID]
	return entry, ok
}

func (idx *OperationIndex) Len() int {
	if idx == nil {
		return 0
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.entries)
}
