package echo

import (
	"errors"
	"net/http"

	echoapi "github.com/labstack/echo/v4"
	breaker "github.com/shuklasaharsh/circuitbreaker"
	"github.com/shuklasaharsh/circuitbreaker/wrapper"
)

type Option func(*config)

type config struct {
	onRejected func(echoapi.Context, error) error
	onError    func(echoapi.Context, error) error
}

func WithOnRejected(fn func(echoapi.Context, error) error) Option {
	return func(cfg *config) {
		if fn != nil {
			cfg.onRejected = fn
		}
	}
}

func WithOnError(fn func(echoapi.Context, error) error) Option {
	return func(cfg *config) {
		if fn != nil {
			cfg.onError = fn
		}
	}
}

func Middleware(reg *wrapper.Registry, breakerName string, opts ...Option) (echoapi.MiddlewareFunc, error) {
	if reg == nil {
		return nil, wrapper.ErrInvalidRegistry
	}
	b, err := reg.Breaker(breakerName)
	if err != nil {
		return nil, err
	}
	return middlewareWithBreaker(b, opts...), nil
}

func MiddlewareWithBreaker(b *breaker.Breaker, opts ...Option) (echoapi.MiddlewareFunc, error) {
	if b == nil {
		return nil, wrapper.ErrInvalidBreaker
	}
	return middlewareWithBreaker(b, opts...), nil
}

func middlewareWithBreaker(b *breaker.Breaker, opts ...Option) echoapi.MiddlewareFunc {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(next echoapi.HandlerFunc) echoapi.HandlerFunc {
		return func(c echoapi.Context) error {
			err := b.ExecuteContext(c.Request().Context(), func() error {
				return next(c)
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
}

func defaultConfig() config {
	return config{
		onRejected: func(c echoapi.Context, _ error) error {
			return c.NoContent(http.StatusServiceUnavailable)
		},
		onError: func(_ echoapi.Context, err error) error {
			return err
		},
	}
}
