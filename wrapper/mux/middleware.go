package mux

import (
	"bufio"
	"errors"
	"net"
	"net/http"

	gorillamux "github.com/gorilla/mux"
	breaker "github.com/shuklasaharsh/circuitbreaker"
	"github.com/shuklasaharsh/circuitbreaker/wrapper"
)

type Option func(*config)

type config struct {
	failureStatusCode int
	onRejected        func(http.ResponseWriter, *http.Request, error)
	onError           func(http.ResponseWriter, *http.Request, error)
}

var errFailureStatus = errors.New("handler returned failure status")

func WithFailureStatusCode(code int) Option {
	return func(cfg *config) {
		if code > 0 {
			cfg.failureStatusCode = code
		}
	}
}

func WithOnRejected(fn func(http.ResponseWriter, *http.Request, error)) Option {
	return func(cfg *config) {
		if fn != nil {
			cfg.onRejected = fn
		}
	}
}

func WithOnError(fn func(http.ResponseWriter, *http.Request, error)) Option {
	return func(cfg *config) {
		if fn != nil {
			cfg.onError = fn
		}
	}
}

func Middleware(reg *wrapper.Registry, breakerName string, opts ...Option) (gorillamux.MiddlewareFunc, error) {
	if reg == nil {
		return nil, wrapper.ErrInvalidRegistry
	}
	b, err := reg.Breaker(breakerName)
	if err != nil {
		return nil, err
	}
	return middlewareWithBreaker(b, opts...), nil
}

func MiddlewareWithBreaker(b *breaker.Breaker, opts ...Option) (gorillamux.MiddlewareFunc, error) {
	if b == nil {
		return nil, wrapper.ErrInvalidBreaker
	}
	return middlewareWithBreaker(b, opts...), nil
}

func middlewareWithBreaker(b *breaker.Breaker, opts ...Option) gorillamux.MiddlewareFunc {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recorder := newStatusRecorder(w)
			err := b.ExecuteContext(r.Context(), func() error {
				next.ServeHTTP(recorder, r)
				if recorder.StatusCode() >= cfg.failureStatusCode {
					return errFailureStatus
				}
				return nil
			})
			if err == nil {
				return
			}
			if errors.Is(err, breaker.ErrCircuitOpen) {
				cfg.onRejected(w, r, err)
				return
			}
			if errors.Is(err, errFailureStatus) {
				return
			}
			cfg.onError(w, r, err)
		})
	}
}

func defaultConfig() config {
	return config{
		failureStatusCode: http.StatusInternalServerError,
		onRejected: func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		},
		onError: func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		},
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func newStatusRecorder(w http.ResponseWriter) *statusRecorder {
	return &statusRecorder{ResponseWriter: w}
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(data)
}

func (r *statusRecorder) StatusCode() int {
	if r.statusCode == 0 {
		return http.StatusOK
	}
	return r.statusCode
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijacker not supported")
	}
	return h.Hijack()
}

func (r *statusRecorder) Push(target string, opts *http.PushOptions) error {
	if p, ok := r.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}
