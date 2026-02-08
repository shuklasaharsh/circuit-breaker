package gin

import (
	"errors"
	"net/http"

	gingonic "github.com/gin-gonic/gin"
	breaker "github.com/shuklasaharsh/circuitbreaker"
	"github.com/shuklasaharsh/circuitbreaker/wrapper"
)

type Option func(*config)

type config struct {
	failureStatusCode int
	onRejected        func(*gingonic.Context, error)
	onError           func(*gingonic.Context, error)
}

var errFailureStatus = errors.New("handler returned failure status")

func WithFailureStatusCode(code int) Option {
	return func(cfg *config) {
		if code > 0 {
			cfg.failureStatusCode = code
		}
	}
}

func WithOnRejected(fn func(*gingonic.Context, error)) Option {
	return func(cfg *config) {
		if fn != nil {
			cfg.onRejected = fn
		}
	}
}

func WithOnError(fn func(*gingonic.Context, error)) Option {
	return func(cfg *config) {
		if fn != nil {
			cfg.onError = fn
		}
	}
}

func Middleware(reg *wrapper.Registry, breakerName string, opts ...Option) (gingonic.HandlerFunc, error) {
	if reg == nil {
		return nil, wrapper.ErrInvalidRegistry
	}
	b, err := reg.Breaker(breakerName)
	if err != nil {
		return nil, err
	}
	return middlewareWithBreaker(b, opts...), nil
}

func MiddlewareWithBreaker(b *breaker.Breaker, opts ...Option) (gingonic.HandlerFunc, error) {
	if b == nil {
		return nil, wrapper.ErrInvalidBreaker
	}
	return middlewareWithBreaker(b, opts...), nil
}

func middlewareWithBreaker(b *breaker.Breaker, opts ...Option) gingonic.HandlerFunc {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(c *gingonic.Context) {
		err := b.ExecuteContext(c.Request.Context(), func() error {
			c.Next()
			if c.Writer.Status() >= cfg.failureStatusCode {
				return errFailureStatus
			}
			return nil
		})
		if err == nil {
			return
		}
		if errors.Is(err, breaker.ErrCircuitOpen) {
			cfg.onRejected(c, err)
			return
		}
		if errors.Is(err, errFailureStatus) {
			return
		}
		cfg.onError(c, err)
	}
}

func defaultConfig() config {
	return config{
		failureStatusCode: http.StatusInternalServerError,
		onRejected: func(c *gingonic.Context, _ error) {
			c.AbortWithStatus(http.StatusServiceUnavailable)
		},
		onError: func(c *gingonic.Context, _ error) {
			c.AbortWithStatus(http.StatusInternalServerError)
		},
	}
}
