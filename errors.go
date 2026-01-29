package breaker

import "github.com/shuklasaharsh/circuitbreaker/errors"

var (
	ErrInvalidThresholdValue = errors.NewError(100, "supplied threshold is invalid", errors.ConfigError)
	ErrInvalidDuration       = errors.NewError(101, "supplied duration is invalid", errors.ConfigError)
	ErrNilFunction           = errors.NewError(102, "function cannot be nil", errors.ConfigError)
)

var (
	ErrCircuitOpen = errors.NewError(100, "circuit breaker is open", errors.CircuitStateError)
)
