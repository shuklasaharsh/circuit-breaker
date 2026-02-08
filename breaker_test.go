package breaker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shuklasaharsh/circuitbreaker/storage"
)

type updateErrorStore struct {
	calls int
	err   error
}

func (s *updateErrorStore) Load(context.Context, string) (storage.Record, error) {
	return storage.Record{}, storage.ErrNotFound
}

func (s *updateErrorStore) Save(context.Context, string, storage.Record) error {
	return nil
}

func (s *updateErrorStore) Update(ctx context.Context, name string, fn func(storage.Record) (storage.Record, error)) (storage.Record, error) {
	s.calls++
	if s.calls == 1 {
		return fn(storage.DefaultRecord())
	}
	return storage.Record{}, s.err
}

func TestBreakerDefaultSnapshot(t *testing.T) {
	b := New("svc")
	record, err := b.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot error: %v", err)
	}
	if record.State != storage.StateClosed {
		t.Fatalf("expected state closed, got %v", record.State)
	}
	if record.Failures != 0 || record.Successes != 0 {
		t.Fatalf("expected zero counts, got failures=%d successes=%d", record.Failures, record.Successes)
	}
}

func TestNewDistributedUsesStore(t *testing.T) {
	ctx := context.Background()
	store := storage.NewMemoryStore()
	expected := storage.Record{
		State:           storage.StateOpen,
		Failures:        3,
		Successes:       1,
		LastFailureTime: time.Now(),
	}
	if err := store.Save(ctx, "svc", expected); err != nil {
		t.Fatalf("save error: %v", err)
	}

	b := NewDistributed("svc", store)
	record, err := b.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot error: %v", err)
	}
	if record.State != expected.State || record.Failures != expected.Failures || record.Successes != expected.Successes {
		t.Fatalf("record mismatch: %#v", record)
	}
}

func TestExecuteNilFunction(t *testing.T) {
	b := New("svc")
	if err := b.Execute(nil); !errors.Is(err, ErrNilFunction) {
		t.Fatalf("expected ErrNilFunction, got %v", err)
	}
}

func TestExecuteOpenCircuit(t *testing.T) {
	b := New("svc",
		WithFailureThreshold(1),
		WithTimeout(time.Minute),
	)
	_ = b.Execute(func() error { return errors.New("boom") })

	err := b.Execute(func() error { return nil })
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestExecuteTransitionsToClosedAfterSuccessThreshold(t *testing.T) {
	timeout := 5 * time.Millisecond
	b := New("svc",
		WithFailureThreshold(1),
		WithSuccessThreshold(2),
		WithTimeout(timeout),
	)
	_ = b.Execute(func() error { return errors.New("boom") })

	time.Sleep(timeout + time.Millisecond)
	if err := b.Execute(func() error { return nil }); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	state, err := b.State(context.Background())
	if err != nil {
		t.Fatalf("state error: %v", err)
	}
	if state != StateHalfOpen {
		t.Fatalf("expected half-open after first success, got %v", state)
	}

	if err := b.Execute(func() error { return nil }); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	state, err = b.State(context.Background())
	if err != nil {
		t.Fatalf("state error: %v", err)
	}
	if state != StateClosed {
		t.Fatalf("expected closed after success threshold, got %v", state)
	}
}

func TestExecuteHalfOpenFailureOpens(t *testing.T) {
	timeout := 5 * time.Millisecond
	b := New("svc",
		WithFailureThreshold(1),
		WithTimeout(timeout),
	)
	_ = b.Execute(func() error { return errors.New("boom") })

	time.Sleep(timeout + time.Millisecond)
	_ = b.Execute(func() error { return errors.New("boom again") })

	state, err := b.State(context.Background())
	if err != nil {
		t.Fatalf("state error: %v", err)
	}
	if state != StateOpen {
		t.Fatalf("expected open after half-open failure, got %v", state)
	}
}

func TestSnapshotNormalizesInvalidState(t *testing.T) {
	ctx := context.Background()
	store := storage.NewMemoryStore()
	invalid := storage.Record{
		State:    storage.State(99),
		Failures: 2,
	}
	if err := store.Save(ctx, "svc", invalid); err != nil {
		t.Fatalf("save error: %v", err)
	}

	b := NewDistributed("svc", store)
	record, err := b.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot error: %v", err)
	}
	if record.State != storage.StateClosed {
		t.Fatalf("expected closed after normalization, got %v", record.State)
	}
}

func TestStateReturnsStoredState(t *testing.T) {
	ctx := context.Background()
	store := storage.NewMemoryStore()
	if err := store.Save(ctx, "svc", storage.Record{State: storage.StateOpen}); err != nil {
		t.Fatalf("save error: %v", err)
	}
	b := NewDistributed("svc", store)
	state, err := b.State(ctx)
	if err != nil {
		t.Fatalf("state error: %v", err)
	}
	if state != StateOpen {
		t.Fatalf("expected open, got %v", state)
	}
}

func TestExecuteJoinsUpdateError(t *testing.T) {
	updateErr := errors.New("update failed")
	store := &updateErrorStore{err: updateErr}
	b := New("svc", WithStorage(store))
	callErr := errors.New("handler failed")

	err := b.Execute(func() error { return callErr })
	if !errors.Is(err, callErr) || !errors.Is(err, updateErr) {
		t.Fatalf("expected joined errors, got %v", err)
	}
}
