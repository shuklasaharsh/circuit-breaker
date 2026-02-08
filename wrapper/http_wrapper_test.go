package wrapper

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	breaker "github.com/shuklasaharsh/circuitbreaker"
)

func TestNewHttpWrapperNilPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic")
		}
	}()
	_ = NewHttpWrapper(nil)
}

func TestHttpWrapperSetBreakerNil(t *testing.T) {
	w := NewHttpWrapper(&http.Client{})
	if err := w.SetBreaker(nil); err != ErrInvalidBreaker {
		t.Fatalf("expected ErrInvalidBreaker, got %v", err)
	}
}

func TestHttpWrapperSetBreakerFromRegistryNil(t *testing.T) {
	w := NewHttpWrapper(&http.Client{})
	if err := w.SetBreakerFromRegistry(nil, "svc"); err != ErrInvalidRegistry {
		t.Fatalf("expected ErrInvalidRegistry, got %v", err)
	}
}

func TestHttpWrapperDoNilRequest(t *testing.T) {
	w := NewHttpWrapper(&http.Client{})
	if _, err := w.Do(nil); err != ErrInvalidRequest {
		t.Fatalf("expected ErrInvalidRequest, got %v", err)
	}
}

func TestHttpWrapperDoWithoutBreaker(t *testing.T) {
	w := NewHttpWrapper(&http.Client{})
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if _, err := w.Do(req); err != ErrInvalidBreaker {
		t.Fatalf("expected ErrInvalidBreaker, got %v", err)
	}
}

func TestHttpWrapperDoSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cb := breaker.New("svc")
	w := NewHttpWrapper(&http.Client{Timeout: time.Second})
	if err := w.SetBreaker(cb); err != nil {
		t.Fatalf("set breaker error: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	resp, err := w.Do(req)
	if err != nil {
		t.Fatalf("do error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHttpWrapperDoCircuitOpen(t *testing.T) {
	cb := breaker.New("svc",
		breaker.WithFailureThreshold(1),
		breaker.WithTimeout(time.Minute),
	)
	_ = cb.Execute(func() error { return errors.New("boom") })

	w := NewHttpWrapper(&http.Client{})
	if err := w.SetBreaker(cb); err != nil {
		t.Fatalf("set breaker error: %v", err)
	}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if _, err := w.Do(req); !errors.Is(err, breaker.ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestHttpWrapperSetBreakerFromRegistry(t *testing.T) {
	reg := NewRegistry()
	cb := breaker.New("svc")
	if err := reg.RegisterBreaker(cb); err != nil {
		t.Fatalf("register error: %v", err)
	}
	w := NewHttpWrapper(&http.Client{})
	if err := w.SetBreakerFromRegistry(reg, "svc"); err != nil {
		t.Fatalf("set breaker error: %v", err)
	}
}
