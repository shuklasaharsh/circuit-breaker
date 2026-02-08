package fiber

import (
	"context"
	"net/http"

	fiberapi "github.com/gofiber/fiber/v2"
	breaker "github.com/shuklasaharsh/circuitbreaker"
	"github.com/shuklasaharsh/circuitbreaker/wrapper"
)

func ExampleMiddleware() {
	reg := wrapper.NewRegistry()
	cb := breaker.New("fiber-api")
	_ = reg.RegisterBreaker(cb)

	app := fiberapi.New()
	middleware, _ := Middleware(reg, "fiber-api", WithContext(func(*fiberapi.Ctx) context.Context {
		return context.Background()
	}))
	app.Use(middleware)

	app.Get("/health", func(c *fiberapi.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})
}
