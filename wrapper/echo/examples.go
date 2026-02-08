package echo

import (
	"net/http"

	echoapi "github.com/labstack/echo/v4"
	breaker "github.com/shuklasaharsh/circuitbreaker"
	"github.com/shuklasaharsh/circuitbreaker/wrapper"
)

func ExampleMiddleware() {
	reg := wrapper.NewRegistry()
	cb := breaker.New("echo-api")
	_ = reg.RegisterBreaker(cb)

	e := echoapi.New()
	middleware, _ := Middleware(reg, "echo-api")
	e.Use(middleware)

	e.GET("/health", func(c echoapi.Context) error {
		return c.NoContent(http.StatusOK)
	})
}
