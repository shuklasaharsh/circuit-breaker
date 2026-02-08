package mux

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gorillamux "github.com/gorilla/mux"
	breaker "github.com/shuklasaharsh/circuitbreaker"
	"github.com/shuklasaharsh/circuitbreaker/storage"
	"github.com/shuklasaharsh/circuitbreaker/wrapper"
)

type errorStore struct {
	err error
}

func (s *errorStore) Load(context.Context, string) (storage.Record, error) {
	return storage.Record{}, s.err
}

func (s *errorStore) Save(context.Context, string, storage.Record) error {
	return s.err
}

func (s *errorStore) Update(context.Context, string, func(storage.Record) (storage.Record, error)) (storage.Record, error) {
	return storage.Record{}, s.err
}

func TestMiddlewareErrors(t *testing.T) {
	if _, err := Middleware(nil, "svc"); err != wrapper.ErrInvalidRegistry {
		t.Fatalf("expected ErrInvalidRegistry, got %v", err)
	}
	reg := wrapper.NewRegistry()
	if _, err := Middleware(reg, "missing"); err != wrapper.ErrBreakerNotFound {
		t.Fatalf("expected ErrBreakerNotFound, got %v", err)
	}
}

func TestMiddlewareWithBreakerNil(t *testing.T) {
	if _, err := MiddlewareWithBreaker(nil); err != wrapper.ErrInvalidBreaker {
		t.Fatalf("expected ErrInvalidBreaker, got %v", err)
	}
}

func TestMiddlewareRejectsOpenCircuit(t *testing.T) {
	cb := breaker.New("svc",
		breaker.WithFailureThreshold(1),
		breaker.WithTimeout(time.Minute),
	)
	_ = cb.Execute(func() error { return errors.New("boom") })

	reg := wrapper.NewRegistry()
	_ = reg.RegisterBreaker(cb)

	rejected := false
	middleware, _ := Middleware(reg, "svc", WithOnRejected(func(w http.ResponseWriter, _ *http.Request, _ error) {
		rejected = true
		http.Error(w, "too many", http.StatusTooManyRequests)
	}))

	router := gorillamux.NewRouter()
	router.Use(middleware)

	called := false
	router.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if !rejected {
		t.Fatalf("expected rejection handler")
	}
	if called {
		t.Fatalf("handler should not run when circuit is open")
	}
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
}

func TestMiddlewareFailureStatusTripsBreaker(t *testing.T) {
	cb := breaker.New("svc", breaker.WithFailureThreshold(1))
	reg := wrapper.NewRegistry()
	_ = reg.RegisterBreaker(cb)

	middleware, _ := Middleware(reg, "svc", WithFailureStatusCode(http.StatusBadRequest))
	router := gorillamux.NewRouter()
	router.Use(middleware)
	router.HandleFunc("/fail", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	req := httptest.NewRequest(http.MethodGet, "/fail", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	state, err := cb.State(context.Background())
	if err != nil {
		t.Fatalf("state error: %v", err)
	}
	if state != breaker.StateOpen {
		t.Fatalf("expected open, got %v", state)
	}
}

func TestMiddlewareOnErrorCalled(t *testing.T) {
	cb := breaker.New("svc", breaker.WithStorage(&errorStore{err: errors.New("update failed")}))
	called := false

	middleware, _ := MiddlewareWithBreaker(cb, WithOnError(func(w http.ResponseWriter, _ *http.Request, _ error) {
		called = true
		http.Error(w, "boom", 520)
	}))

	router := gorillamux.NewRouter()
	router.Use(middleware)
	router.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if !called {
		t.Fatalf("expected onError handler")
	}
	if rec.Code != 520 {
		t.Fatalf("expected 520, got %d", rec.Code)
	}
}
