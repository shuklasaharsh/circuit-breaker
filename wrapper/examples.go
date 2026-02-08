package wrapper

import (
	"net/http"
	"net/http/httptest"
	"time"

	breaker "github.com/shuklasaharsh/circuitbreaker"
)

func ExampleHTTPWrapper() {
	reg := NewRegistry()
	cb := breaker.New("http-client")
	_ = reg.RegisterBreaker(cb)

	client := &http.Client{Timeout: 2 * time.Second}
	wrapper := NewHttpWrapper(client)
	_ = wrapper.SetBreakerFromRegistry(reg, "http-client")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	_, _ = wrapper.Do(req)
}
