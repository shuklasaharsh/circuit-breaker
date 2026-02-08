package storage

import (
	"context"
	"sync"
)

type MemoryStore struct {
	mu      sync.RWMutex
	records map[string]Record
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		records: make(map[string]Record),
	}
}

func (m *MemoryStore) Load(_ context.Context, name string) (Record, error) {
	m.mu.RLock()
	record, ok := m.records[name]
	m.mu.RUnlock()
	if !ok {
		return Record{}, ErrNotFound
	}
	return record, nil
}

func (m *MemoryStore) Save(_ context.Context, name string, record Record) error {
	m.mu.Lock()
	m.records[name] = record
	m.mu.Unlock()
	return nil
}

func (m *MemoryStore) Update(_ context.Context, name string, fn func(Record) (Record, error)) (Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	record, ok := m.records[name]
	if !ok {
		record = DefaultRecord()
	}

	updated, err := fn(record)
	if err != nil {
		return Record{}, err
	}
	m.records[name] = updated
	return updated, nil
}
