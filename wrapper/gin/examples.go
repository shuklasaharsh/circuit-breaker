package gin

import (
	"net/http"

	gingonic "github.com/gin-gonic/gin"
	breaker "github.com/shuklasaharsh/circuitbreaker"
	"github.com/shuklasaharsh/circuitbreaker/wrapper"
)

func ExampleMiddleware() {
	reg := wrapper.NewRegistry()
	cb := breaker.New("gin-api")
	_ = reg.RegisterBreaker(cb)

	router := gingonic.New()
	middleware, _ := Middleware(reg, "gin-api")
	router.Use(middleware)

	router.GET("/health", func(c *gingonic.Context) {
		c.Status(http.StatusOK)
	})
}
