# Circuit Breaker

A lightweight, extensible circuit breaker library for Go with first-class support for `net/http` and popular frameworks.

## Planned Features

- **Framework Agnostic** — Works with `net/http`, Gin, Fiber, and FastHTTP out of the box
- **Built-in Breakers** — Ready-to-use circuit breakers for HTTP clients, Redis, and SQL databases
- **Extensible Health Checks** — Plug in custom health checkers for any backend
- **Prometheus Metrics** — Built-in metrics server for observability
- **Sensible Defaults** — Works out of the box with production-ready settings

## Installation

```bash
go get github.com/shuklasaharsh/circuit-breaker
```

## Quick Start

```go
package main

import (
    "net/http"
    "github.com/shuklasaharsh/circuit-breaker"
)

func main() {
    // Create a breaker with default settings
    cb := circuitbreaker.New("my-service",
        circuitbreaker.WithThreshold(5),
        circuitbreaker.WithTimeout(30 * time.Second),
    )

    // Wrap your HTTP client
    client := cb.WrapClient(http.DefaultClient)

    // Use it normally
    resp, err := client.Get("https://api.example.com/data")
}
```

## Usage with Frameworks

### Gin

```go
router := gin.Default()
router.Use(circuitbreaker.GinMiddleware(cb))
```

### Fiber

```go
app := fiber.New()
app.Use(circuitbreaker.FiberMiddleware(cb))
```

### FastHTTP

```go
handler := circuitbreaker.FastHTTPMiddleware(cb, yourHandler)
```

## Built-in Breakers

### HTTP

```go
httpBreaker := circuitbreaker.NewHTTPBreaker("api-service",
    circuitbreaker.WithHealthCheck(circuitbreaker.HTTPHealthCheck("https://api.example.com/health")),
)
```

### Redis

```go
redisBreaker := circuitbreaker.NewRedisBreaker("redis-primary",
    circuitbreaker.WithRedisAddr("localhost:6379"),
)
```

### SQL

```go
sqlBreaker := circuitbreaker.NewSQLBreaker("postgres-main",
    circuitbreaker.WithDB(db),
    circuitbreaker.WithPingTimeout(2 * time.Second),
)
```

## Custom Health Checkers

```go
checker := circuitbreaker.HealthCheckerFunc(func(ctx context.Context) error {
    // Your custom health check logic
    return nil
})

cb := circuitbreaker.New("my-service",
    circuitbreaker.WithHealthCheck(checker),
)
```

## Prometheus Metrics

```go
// Start the metrics server
metrics := circuitbreaker.NewMetricsServer(":9090")
metrics.Register(cb)
go metrics.ListenAndServe()
```

Exposed metrics:

| Metric | Type | Description |
|--------|------|-------------|
| `circuitbreaker_state` | Gauge | Current state (0=closed, 1=open, 2=half-open) |
| `circuitbreaker_requests_total` | Counter | Total requests by outcome |
| `circuitbreaker_failures_total` | Counter | Total failures |
| `circuitbreaker_state_transitions_total` | Counter | State transitions |
| `circuitbreaker_latency_seconds` | Histogram | Request latency |

## Configuration

| Option | Default | Description |
|--------|---------|-------------|
| `WithThreshold(n)` | 5 | Failures before opening |
| `WithTimeout(d)` | 60s | Time before half-open |
| `WithSuccessThreshold(n)` | 2 | Successes to close from half-open |
| `WithHealthCheck(hc)` | nil | Custom health checker |
| `WithOnStateChange(fn)` | nil | State change callback |

## Circuit States

```
     ┌─────────────────────────────────────┐
     │                                     │
     ▼                                     │
┌─────────┐  failure threshold  ┌──────┐   │ success threshold
│ CLOSED  │ ─────────────────▶  │ OPEN │   │
└─────────┘                     └──────┘   │
     ▲                              │      │
     │                              │ timeout
     │                              ▼      │
     │                        ┌───────────┐│
     │    failure             │ HALF-OPEN ├┘
     └────────────────────────┴───────────┘
```

## License

MIT
