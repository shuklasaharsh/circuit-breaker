package breaker

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/shuklasaharsh/circuitbreaker/storage"
)

// Breaker
type Breaker struct {
	Name             string
	failureThreshold int64
	successThreshold int64
	timeout          time.Duration
	store            storage.Store
}

func New(name string, opts ...Option) *Breaker {
	cfg := defaultConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.Store == nil {
		cfg.Store = storage.NewMemoryStore()
	}

	return &Breaker{
		Name:             name,
		failureThreshold: cfg.FailureThreshold,
		successThreshold: cfg.SuccessThreshold,
		timeout:          cfg.Timeout,
		store:            cfg.Store,
	}
}

// Execute runs the given function through the circuit breaker
func (b *Breaker) Execute(fn func() error) error {
	return b.ExecuteContext(context.Background(), fn)
}

// ExecuteContext runs the given function through the circuit breaker with a context
func (b *Breaker) ExecuteContext(ctx context.Context, fn func() error) error {
	// Validation
	if fn == nil {
		return ErrNilFunction
	}

	// Check if we can execute
	allowed, err := b.allow(ctx)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrCircuitOpen
	}

	// Execute the function
	err = fn()

	// Record the result
	if err == nil {
		return b.onSuccess(ctx)
	}
	recordErr := b.onFailure(ctx)
	if recordErr != nil {
		return stderrors.Join(err, recordErr)
	}
	return err
}

// NewDistributed returns a breaker backed by a shared storage engine.
func NewDistributed(name string, store storage.Store, opts ...Option) *Breaker {
	opts = append(opts, WithStorage(store))
	return New(name, opts...)
}

// Snapshot returns the current breaker record.
func (b *Breaker) Snapshot(ctx context.Context) (storage.Record, error) {
	record, err := b.store.Load(ctx, b.Name)
	if err == nil {
		return normalizeRecord(record), nil
	}
	if stderrors.Is(err, storage.ErrNotFound) {
		return storage.DefaultRecord(), nil
	}
	return storage.Record{}, err
}

// State returns the current circuit state.
func (b *Breaker) State(ctx context.Context) (State, error) {
	record, err := b.Snapshot(ctx)
	if err != nil {
		return StateClosed, err
	}
	return record.State, nil
}

// allow checks if a request can be executed and performs state transitions.
func (b *Breaker) allow(ctx context.Context) (bool, error) {
	var allowed bool
	_, err := b.store.Update(ctx, b.Name, func(record storage.Record) (storage.Record, error) {
		record = normalizeRecord(record)
		switch record.State {
		case storage.StateClosed:
			allowed = true
			return record, nil
		case storage.StateOpen:
			if time.Since(record.LastFailureTime) > b.timeout {
				record.State = storage.StateHalfOpen
				record.Successes = 0
				allowed = true
			} else {
				allowed = false
			}
			return record, nil
		case storage.StateHalfOpen:
			allowed = true
			return record, nil
		default:
			allowed = false
			return storage.DefaultRecord(), nil
		}
	})
	return allowed, err
}

// onSuccess handles a successful execution.
func (b *Breaker) onSuccess(ctx context.Context) error {
	_, err := b.store.Update(ctx, b.Name, func(record storage.Record) (storage.Record, error) {
		record = normalizeRecord(record)
		switch record.State {
		case storage.StateClosed:
			record.Failures = 0
			record.Successes = 0
		case storage.StateHalfOpen:
			record.Successes++
			if record.Successes >= b.successThreshold {
				record.State = storage.StateClosed
				record.Failures = 0
				record.Successes = 0
			}
		}
		return record, nil
	})
	return err
}

// onFailure handles a failed execution.
func (b *Breaker) onFailure(ctx context.Context) error {
	now := time.Now()
	_, err := b.store.Update(ctx, b.Name, func(record storage.Record) (storage.Record, error) {
		record = normalizeRecord(record)
		record.LastFailureTime = now

		switch record.State {
		case storage.StateClosed:
			record.Failures++
			record.Successes = 0
			if record.Failures >= b.failureThreshold {
				record.State = storage.StateOpen
			}
		case storage.StateHalfOpen:
			record.State = storage.StateOpen
			record.Successes = 0
		}
		return record, nil
	})
	return err
}

func normalizeRecord(record storage.Record) storage.Record {
	switch record.State {
	case storage.StateClosed, storage.StateOpen, storage.StateHalfOpen:
		return record
	default:
		return storage.DefaultRecord()
	}
}
