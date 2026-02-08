package wrapper

import (
	"net/http"

	breaker "github.com/shuklasaharsh/circuitbreaker"
)

type HttpWrapper struct {
	httpClient *http.Client
	breaker    *breaker.Breaker
}

func NewHttpWrapper(httpClient *http.Client) *HttpWrapper {
	if httpClient == nil {
		panic(ErrInvalidHttpClient)
	}
	return &HttpWrapper{httpClient: httpClient}
}

func (w *HttpWrapper) SetBreaker(b *breaker.Breaker) error {
	if b == nil {
		return ErrInvalidBreaker
	}
	w.breaker = b
	return nil
}

func (w *HttpWrapper) SetBreakerFromRegistry(reg *Registry, name string) error {
	if reg == nil {
		return ErrInvalidRegistry
	}
	b, err := reg.Breaker(name)
	if err != nil {
		return err
	}
	w.breaker = b
	return nil
}

func (w *HttpWrapper) Do(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, ErrInvalidRequest
	}
	if w.breaker == nil {
		return nil, ErrInvalidBreaker
	}

	var resp *http.Response
	err := w.breaker.ExecuteContext(req.Context(), func() error {
		var err error
		resp, err = w.httpClient.Do(req)
		return err
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
