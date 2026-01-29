package breaker

import (
	"sync"
	"time"
)

// Breaker
type Breaker struct {
	Name            string
	State           State
	failures        int64
	successes       int64
	lastFailureTime time.Time

	failureThreshold int64
	successThreshold int64
	timeout          time.Duration

	mu sync.RWMutex
}

func New(name string, opts ...Option) *Breaker {
	cfg := defaultConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	return &Breaker{
		Name:             name,
		State:            StateClosed,
		failures:         0,
		successes:        0,
		lastFailureTime:  time.Time{},
		failureThreshold: cfg.FailureThreshold,
		successThreshold: cfg.SuccessThreshold,
		timeout:          cfg.Timeout,
		mu:               sync.RWMutex{},
	}
}

// Execute runs the given function through the circuit breaker
func (b *Breaker) Execute(fn func() error) error {
	// Validation
	if fn == nil {
		return ErrNilFunction
	}

	// Check if we can execute
	if !b.canExecute() {
		return ErrCircuitOpen
	}

	// Execute the function
	err := fn()

	// Record the result
	b.recordResult(err)

	return err
}

// canExecute checks if a request can be executed
func (b *Breaker) canExecute() bool {
	switch b.State {
	case StateClosed:
		return true

	case StateOpen:
		// Check if timeout has elapsed
		if time.Since(b.lastFailureTime) > b.timeout {
			// Transition to half-open
			b.State = StateHalfOpen
			b.successes = 0
			return true
		}
		return false

	case StateHalfOpen:
		return true

	default:
		return false
	}
}

// onSuccess handles a successful execution
func (b *Breaker) onSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.State {
	case StateClosed:
		// Reset failure count on success
		b.failures = 0
	case StateHalfOpen:
		// Increment success count
		b.successes++
		// If we've reached success threshold, close the circuit
		if b.successes >= b.successThreshold {
			b.State = StateClosed
			b.failures = 0
			b.successes = 0
		}
	}
}

// onFailure handles a failed execution
func (b *Breaker) onFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lastFailureTime = time.Now()

	switch b.State {
	case StateClosed:
		b.failures++

		// Open circuit if threshold reached
		if b.failures >= b.failureThreshold {
			b.State = StateOpen
		}

	case StateHalfOpen:
		// Any failure in half-open immediately opens circuit
		b.State = StateOpen
		b.successes = 0
	}
}

// recordResult records the outcome of an execution
func (b *Breaker) recordResult(err error) {
	if err == nil {
		b.onSuccess()
	} else {
		b.onFailure()
	}
}
