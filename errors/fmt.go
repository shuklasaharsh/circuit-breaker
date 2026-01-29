package errors

type ErrorFmt string

const (
	ConfigError       ErrorFmt = "configuration error"
	CircuitStateError ErrorFmt = "circuit state error"
)
