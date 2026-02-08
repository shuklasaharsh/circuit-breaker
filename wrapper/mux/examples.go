package mux

import (
	"net/http"

	gorillamux "github.com/gorilla/mux"
	breaker "github.com/shuklasaharsh/circuitbreaker"
	"github.com/shuklasaharsh/circuitbreaker/wrapper"
)

func ExampleMiddleware() {
	reg := wrapper.NewRegistry()
	cb := breaker.New("mux-api")
	_ = reg.RegisterBreaker(cb)

	router := gorillamux.NewRouter()
	middleware, _ := Middleware(reg, "mux-api")
	router.Use(middleware)

	router.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
