package storage

import (
	"context"
	"errors"
	"testing"
)

func TestMemoryStoreLoadMissing(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.Load(context.Background(), "svc")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStoreSaveAndLoad(t *testing.T) {
	store := NewMemoryStore()
	record := Record{State: StateOpen, Failures: 2}
	if err := store.Save(context.Background(), "svc", record); err != nil {
		t.Fatalf("save error: %v", err)
	}
	loaded, err := store.Load(context.Background(), "svc")
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if loaded.State != record.State || loaded.Failures != record.Failures {
		t.Fatalf("expected %#v, got %#v", record, loaded)
	}
}

func TestMemoryStoreUpdateCreatesDefault(t *testing.T) {
	store := NewMemoryStore()
	updated, err := store.Update(context.Background(), "svc", func(r Record) (Record, error) {
		if r.State != StateClosed {
			t.Fatalf("expected default state closed, got %v", r.State)
		}
		r.Failures++
		return r, nil
	})
	if err != nil {
		t.Fatalf("update error: %v", err)
	}
	if updated.Failures != 1 {
		t.Fatalf("expected failures 1, got %d", updated.Failures)
	}
}

func TestMemoryStoreUpdateErrorDoesNotPersist(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.Update(context.Background(), "svc", func(r Record) (Record, error) {
		return Record{}, errors.New("boom")
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	_, err = store.Load(context.Background(), "svc")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
