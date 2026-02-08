package breaker

import (
	"time"

	"github.com/shuklasaharsh/circuitbreaker/storage"
)

type Config struct {
	FailureThreshold int64
	SuccessThreshold int64
	Timeout          time.Duration
	Store            storage.Store
}

type Option func(*Config)

func WithFailureThreshold(n int64) Option {
	if n <= 0 {
		panic(ErrInvalidThresholdValue)
	}
	return func(c *Config) {
		c.FailureThreshold = n
	}
}

func WithSuccessThreshold(n int64) Option {
	if n <= 0 {
		panic(ErrInvalidThresholdValue)
	}
	return func(c *Config) {
		c.SuccessThreshold = n
	}
}

func WithTimeout(d time.Duration) Option {
	if d <= 0 {
		panic(ErrInvalidDuration)
	}

	return func(c *Config) {
		c.Timeout = d
	}
}

func WithStorage(store storage.Store) Option {
	if store == nil {
		panic(ErrInvalidStorage)
	}
	return func(c *Config) {
		c.Store = store
	}
}

func defaultConfig() Config {
	return Config{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          60 * time.Second,
		Store:            storage.NewMemoryStore(),
	}
}
