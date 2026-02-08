package wrapper

import (
	"testing"

	breaker "github.com/shuklasaharsh/circuitbreaker"
)

func TestRegistryRegisterAndLookup(t *testing.T) {
	reg := NewRegistry()
	cb := breaker.New("svc")
	if err := reg.RegisterBreaker(cb); err != nil {
		t.Fatalf("register error: %v", err)
	}
	got, err := reg.Breaker("svc")
	if err != nil {
		t.Fatalf("lookup error: %v", err)
	}
	if got != cb {
		t.Fatalf("expected breaker pointer match")
	}
}

func TestRegistryNilReceiver(t *testing.T) {
	var reg *Registry
	cb := breaker.New("svc")
	if err := reg.RegisterBreaker(cb); err != ErrInvalidRegistry {
		t.Fatalf("expected ErrInvalidRegistry, got %v", err)
	}
}

func TestRegistryInvalidBreaker(t *testing.T) {
	reg := NewRegistry()
	if err := reg.RegisterBreaker(nil); err != ErrInvalidBreaker {
		t.Fatalf("expected ErrInvalidBreaker, got %v", err)
	}
}

func TestRegistryInvalidBreakerName(t *testing.T) {
	reg := NewRegistry()
	cb := breaker.New("svc")
	if err := reg.RegisterBreakerWithName("", cb); err != ErrInvalidBreakerName {
		t.Fatalf("expected ErrInvalidBreakerName, got %v", err)
	}
}

func TestRegistryBreakerNotFound(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.Breaker("missing")
	if err != ErrBreakerNotFound {
		t.Fatalf("expected ErrBreakerNotFound, got %v", err)
	}
}

func TestRegistryBreakerInvalidName(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.Breaker("")
	if err != ErrInvalidBreakerName {
		t.Fatalf("expected ErrInvalidBreakerName, got %v", err)
	}
}
