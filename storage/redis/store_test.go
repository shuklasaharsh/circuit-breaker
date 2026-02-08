package redis

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/shuklasaharsh/circuitbreaker/storage"
)

type mockClient struct {
	mu        sync.Mutex
	data      map[string]string
	conflicts int
}

type mockTx struct {
	client *mockClient
}

func newMockClient() *mockClient {
	return &mockClient{data: make(map[string]string)}
}

func (m *mockClient) Get(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, ok := m.data[key]
	if !ok {
		return "", storage.ErrNotFound
	}
	return value, nil
}

func (m *mockClient) Set(_ context.Context, key, value string, _ time.Duration) error {
	m.mu.Lock()
	m.data[key] = value
	m.mu.Unlock()
	return nil
}

func (m *mockClient) Watch(ctx context.Context, key string, fn func(Tx) error) error {
	m.mu.Lock()
	if m.conflicts > 0 {
		m.conflicts--
		m.mu.Unlock()
		return storage.ErrConflict
	}
	m.mu.Unlock()
	return fn(&mockTx{client: m})
}

func (t *mockTx) Get(ctx context.Context, key string) (string, error) {
	return t.client.Get(ctx, key)
}

func (t *mockTx) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return t.client.Set(ctx, key, value, ttl)
}

type noopCodec struct{}

func (noopCodec) Marshal(storage.Record) ([]byte, error) {
	return []byte(`{}`), nil
}

func (noopCodec) Unmarshal([]byte) (storage.Record, error) {
	return storage.DefaultRecord(), nil
}

type errorCodec struct{}

func (errorCodec) Marshal(storage.Record) ([]byte, error) {
	return nil, errors.New("marshal failed")
}

func (errorCodec) Unmarshal([]byte) (storage.Record, error) {
	return storage.Record{}, errors.New("unmarshal failed")
}

func TestNewNilClient(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewOptions(t *testing.T) {
	store, err := New(newMockClient(),
		WithKeyPrefix("cb"),
		WithTTL(2*time.Minute),
		WithMaxRetries(5),
		WithCodec(noopCodec{}),
	)
	if err != nil {
		t.Fatalf("new error: %v", err)
	}
	if store.keyPrefix != "cb" {
		t.Fatalf("expected key prefix cb, got %q", store.keyPrefix)
	}
	if store.ttl != 2*time.Minute {
		t.Fatalf("expected ttl 2m, got %v", store.ttl)
	}
	if store.maxRetries != 5 {
		t.Fatalf("expected maxRetries 5, got %d", store.maxRetries)
	}
	if _, ok := store.codec.(noopCodec); !ok {
		t.Fatalf("expected noopCodec, got %T", store.codec)
	}
}

func TestWithMaxRetriesNegative(t *testing.T) {
	store, err := New(newMockClient(), WithMaxRetries(-1))
	if err != nil {
		t.Fatalf("new error: %v", err)
	}
	if store.maxRetries != 0 {
		t.Fatalf("expected maxRetries 0, got %d", store.maxRetries)
	}
}

func TestLoadNotFound(t *testing.T) {
	store, err := New(newMockClient())
	if err != nil {
		t.Fatalf("new error: %v", err)
	}
	_, err = store.Load(context.Background(), "missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSaveAndLoad(t *testing.T) {
	store, err := New(newMockClient())
	if err != nil {
		t.Fatalf("new error: %v", err)
	}
	record := storage.Record{State: storage.StateOpen, Failures: 1}
	if err := store.Save(context.Background(), "svc", record); err != nil {
		t.Fatalf("save error: %v", err)
	}
	loaded, err := store.Load(context.Background(), "svc")
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if loaded.State != record.State || loaded.Failures != record.Failures {
		t.Fatalf("record mismatch: %#v", loaded)
	}
}

func TestSaveCodecError(t *testing.T) {
	store, err := New(newMockClient(), WithCodec(errorCodec{}))
	if err != nil {
		t.Fatalf("new error: %v", err)
	}
	err = store.Save(context.Background(), "svc", storage.Record{})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestUpdateSuccess(t *testing.T) {
	store, err := New(newMockClient())
	if err != nil {
		t.Fatalf("new error: %v", err)
	}
	updated, err := store.Update(context.Background(), "svc", func(r storage.Record) (storage.Record, error) {
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

func TestUpdateConflictRetries(t *testing.T) {
	client := newMockClient()
	client.conflicts = 1
	store, err := New(client, WithMaxRetries(2))
	if err != nil {
		t.Fatalf("new error: %v", err)
	}
	_, err = store.Update(context.Background(), "svc", func(r storage.Record) (storage.Record, error) {
		r.Successes++
		return r, nil
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestUpdateConflictExhausted(t *testing.T) {
	client := newMockClient()
	client.conflicts = 2
	store, err := New(client, WithMaxRetries(1))
	if err != nil {
		t.Fatalf("new error: %v", err)
	}
	_, err = store.Update(context.Background(), "svc", func(r storage.Record) (storage.Record, error) {
		r.Successes++
		return r, nil
	})
	if !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestLoadFromTxMissing(t *testing.T) {
	store, err := New(newMockClient())
	if err != nil {
		t.Fatalf("new error: %v", err)
	}
	record, err := store.loadFromTx(context.Background(), &mockTx{client: newMockClient()}, "missing")
	if err != nil {
		t.Fatalf("loadFromTx error: %v", err)
	}
	if record.State != storage.StateClosed {
		t.Fatalf("expected default record, got %v", record.State)
	}
}

func TestKeyPrefix(t *testing.T) {
	store, err := New(newMockClient(), WithKeyPrefix("cb"))
	if err != nil {
		t.Fatalf("new error: %v", err)
	}
	if store.key("svc") != "cb:svc" {
		t.Fatalf("unexpected key: %s", store.key("svc"))
	}
}
