package fiber

import (
	"context"
	"errors"
	"net/http"

	fiberapi "github.com/gofiber/fiber/v2"
	breaker "github.com/shuklasaharsh/circuitbreaker"
	"github.com/shuklasaharsh/circuitbreaker/wrapper"
)

type Option func(*config)

type ContextFunc func(*fiberapi.Ctx) context.Context

type config struct {
	contextFunc ContextFunc
	onRejected  func(*fiberapi.Ctx, error) error
	onError     func(*fiberapi.Ctx, error) error
}

func WithContext(fn ContextFunc) Option {
	return func(cfg *config) {
		if fn != nil {
			cfg.contextFunc = fn
		}
	}
}

func WithOnRejected(fn func(*fiberapi.Ctx, error) error) Option {
	return func(cfg *config) {
		if fn != nil {
			cfg.onRejected = fn
		}
	}
}

func WithOnError(fn func(*fiberapi.Ctx, error) error) Option {
	return func(cfg *config) {
		if fn != nil {
			cfg.onError = fn
		}
	}
}

func Middleware(reg *wrapper.Registry, breakerName string, opts ...Option) (fiberapi.Handler, error) {
	if reg == nil {
		return nil, wrapper.ErrInvalidRegistry
	}
	b, err := reg.Breaker(breakerName)
	if err != nil {
		return nil, err
	}
	return middlewareWithBreaker(b, opts...), nil
}

func MiddlewareWithBreaker(b *breaker.Breaker, opts ...Option) (fiberapi.Handler, error) {
	if b == nil {
		return nil, wrapper.ErrInvalidBreaker
	}
	return middlewareWithBreaker(b, opts...), nil
}

func middlewareWithBreaker(b *breaker.Breaker, opts ...Option) fiberapi.Handler {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(c *fiberapi.Ctx) error {
		ctx := cfg.contextFunc(c)
		if ctx == nil {
			ctx = context.Background()
		}
		err := b.ExecuteContext(ctx, func() error {
			return c.Next()
		})
		if err == nil {
			return nil
		}
		if errors.Is(err, breaker.ErrCircuitOpen) {
			return cfg.onRejected(c, err)
		}
		return cfg.onError(c, err)
	}
}

func defaultConfig() config {
	return config{
		contextFunc: func(_ *fiberapi.Ctx) context.Context {
			return context.Background()
		},
		onRejected: func(c *fiberapi.Ctx, _ error) error {
			return c.SendStatus(http.StatusServiceUnavailable)
		},
		onError: func(_ *fiberapi.Ctx, err error) error {
			return err
		},
	}
}
