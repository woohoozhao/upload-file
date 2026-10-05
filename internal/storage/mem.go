package storage

import (
	"context"
	"io"
	"sync"
)

// MemStorer is an in-memory Storer, useful for tests and demos.
//
// All bytes read from src are buffered in memory under the given key.
// Not safe for production use.
type MemStorer struct {
	mu    sync.Mutex
	files map[string][]byte
}

func NewMemStorer() *MemStorer {
	return &MemStorer{files: make(map[string][]byte)}
}

func (m *MemStorer) Put(_ context.Context, key string, src io.Reader, _ int64, _ string) (string, error) {
	data, err := io.ReadAll(src)
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	m.files[key] = data
	m.mu.Unlock()
	return key, nil
}

// Get returns the bytes stored under key (and whether the key exists).
func (m *MemStorer) Get(key string) ([]byte, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.files[key]
	return d, ok
}

// Len returns the number of stored keys.
func (m *MemStorer) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.files)
}

// Keys returns a snapshot of all stored keys, in unspecified order.
func (m *MemStorer) Keys() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.files))
	for k := range m.files {
		out = append(out, k)
	}
	return out
}
