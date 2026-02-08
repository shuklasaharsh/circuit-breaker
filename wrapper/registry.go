package wrapper

import (
	"sync"

	breaker "github.com/shuklasaharsh/circuitbreaker"
)

type Registry struct {
	mu       sync.RWMutex
	breakers map[string]*breaker.Breaker
}

func NewRegistry() *Registry {
	return &Registry{
		breakers: make(map[string]*breaker.Breaker),
	}
}

func (r *Registry) RegisterBreaker(b *breaker.Breaker) error {
	if b == nil {
		return ErrInvalidBreaker
	}
	return r.RegisterBreakerWithName(b.Name, b)
}

func (r *Registry) RegisterBreakerWithName(name string, b *breaker.Breaker) error {
	if r == nil {
		return ErrInvalidRegistry
	}
	if b == nil {
		return ErrInvalidBreaker
	}
	if name == "" {
		return ErrInvalidBreakerName
	}

	r.mu.Lock()
	if r.breakers == nil {
		r.breakers = make(map[string]*breaker.Breaker)
	}
	r.breakers[name] = b
	r.mu.Unlock()
	return nil
}

func (r *Registry) Breaker(name string) (*breaker.Breaker, error) {
	if r == nil {
		return nil, ErrInvalidRegistry
	}
	if name == "" {
		return nil, ErrInvalidBreakerName
	}

	r.mu.RLock()
	b := r.breakers[name]
	r.mu.RUnlock()
	if b == nil {
		return nil, ErrBreakerNotFound
	}
	return b, nil
}
