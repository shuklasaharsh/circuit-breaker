package wrapper

import "github.com/shuklasaharsh/circuitbreaker/errors"

var (
	ErrInvalidHttpClient  = errors.NewError(100, "http client cannot be nil", errors.ConfigError)
	ErrInvalidBreaker     = errors.NewError(101, "breaker cannot be nil", errors.ConfigError)
	ErrInvalidRegistry    = errors.NewError(102, "registry cannot be nil", errors.ConfigError)
	ErrInvalidBreakerName = errors.NewError(103, "breaker name cannot be empty", errors.ConfigError)
	ErrBreakerNotFound    = errors.NewError(104, "breaker not found in registry", errors.ConfigError)
	ErrInvalidRequest     = errors.NewError(105, "http request cannot be nil", errors.ConfigError)
)
