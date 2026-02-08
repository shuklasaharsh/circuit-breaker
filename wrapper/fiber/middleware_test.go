package fiber

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	fiberapi "github.com/gofiber/fiber/v2"
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
	middleware, _ := Middleware(reg, "svc", WithOnRejected(func(c *fiberapi.Ctx, _ error) error {
		rejected = true
		return c.SendStatus(http.StatusTooManyRequests)
	}))

	app := fiberapi.New()
	app.Use(middleware)

	called := false
	app.Get("/health", func(c *fiberapi.Ctx) error {
		called = true
		return c.SendStatus(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if !rejected {
		t.Fatalf("expected rejection handler")
	}
	if called {
		t.Fatalf("handler should not run when circuit is open")
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", resp.StatusCode)
	}
}

func TestMiddlewareHandlerErrorTripsBreaker(t *testing.T) {
	cb := breaker.New("svc", breaker.WithFailureThreshold(1))
	reg := wrapper.NewRegistry()
	_ = reg.RegisterBreaker(cb)

	middleware, _ := Middleware(reg, "svc")
	app := fiberapi.New()
	app.Use(middleware)
	app.Get("/fail", func(c *fiberapi.Ctx) error {
		return errors.New("fail")
	})

	req := httptest.NewRequest(http.MethodGet, "/fail", nil)
	if _, err := app.Test(req, -1); err != nil {
		t.Fatalf("request error: %v", err)
	}

	state, err := cb.State(context.Background())
	if err != nil {
		t.Fatalf("state error: %v", err)
	}
	if state != breaker.StateOpen {
		t.Fatalf("expected open, got %v", state)
	}
}

func TestMiddlewareWithContextCalled(t *testing.T) {
	cb := breaker.New("svc")
	reg := wrapper.NewRegistry()
	_ = reg.RegisterBreaker(cb)

	called := false
	middleware, _ := Middleware(reg, "svc", WithContext(func(*fiberapi.Ctx) context.Context {
		called = true
		return context.Background()
	}))

	app := fiberapi.New()
	app.Use(middleware)
	app.Get("/health", func(c *fiberapi.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	if _, err := app.Test(req, -1); err != nil {
		t.Fatalf("request error: %v", err)
	}
	if !called {
		t.Fatalf("expected context func to be called")
	}
}

func TestMiddlewareOnErrorCalled(t *testing.T) {
	cb := breaker.New("svc", breaker.WithStorage(&errorStore{err: errors.New("update failed")}))
	called := false

	middleware, _ := MiddlewareWithBreaker(cb, WithOnError(func(c *fiberapi.Ctx, _ error) error {
		called = true
		return c.SendStatus(520)
	}))

	app := fiberapi.New()
	app.Use(middleware)
	app.Get("/health", func(c *fiberapi.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if !called {
		t.Fatalf("expected onError handler")
	}
	if resp.StatusCode != 520 {
		t.Fatalf("expected 520, got %d", resp.StatusCode)
	}
}
