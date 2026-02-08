package breaker

import (
	"testing"
	"time"

	"github.com/shuklasaharsh/circuitbreaker/storage"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.FailureThreshold != 5 {
		t.Fatalf("expected failure threshold 5, got %d", cfg.FailureThreshold)
	}
	if cfg.SuccessThreshold != 2 {
		t.Fatalf("expected success threshold 2, got %d", cfg.SuccessThreshold)
	}
	if cfg.Timeout != 60*time.Second {
		t.Fatalf("expected timeout 60s, got %v", cfg.Timeout)
	}
	if cfg.Store == nil {
		t.Fatalf("expected default store")
	}
}

func TestWithFailureThreshold(t *testing.T) {
	cfg := defaultConfig()
	WithFailureThreshold(3)(&cfg)
	if cfg.FailureThreshold != 3 {
		t.Fatalf("expected failure threshold 3, got %d", cfg.FailureThreshold)
	}
}

func TestWithFailureThresholdPanics(t *testing.T) {
	assertPanics(t, func() { _ = WithFailureThreshold(0) })
}

func TestWithSuccessThreshold(t *testing.T) {
	cfg := defaultConfig()
	WithSuccessThreshold(4)(&cfg)
	if cfg.SuccessThreshold != 4 {
		t.Fatalf("expected success threshold 4, got %d", cfg.SuccessThreshold)
	}
}

func TestWithSuccessThresholdPanics(t *testing.T) {
	assertPanics(t, func() { _ = WithSuccessThreshold(0) })
}

func TestWithTimeout(t *testing.T) {
	cfg := defaultConfig()
	WithTimeout(10 * time.Second)(&cfg)
	if cfg.Timeout != 10*time.Second {
		t.Fatalf("expected timeout 10s, got %v", cfg.Timeout)
	}
}

func TestWithTimeoutPanics(t *testing.T) {
	assertPanics(t, func() { _ = WithTimeout(0) })
}

func TestWithStorage(t *testing.T) {
	cfg := defaultConfig()
	store := storage.NewMemoryStore()
	WithStorage(store)(&cfg)
	if cfg.Store != store {
		t.Fatalf("expected custom store")
	}
}

func TestWithStoragePanics(t *testing.T) {
	assertPanics(t, func() { _ = WithStorage(nil) })
}

func assertPanics(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic")
		}
	}()
	fn()
}
