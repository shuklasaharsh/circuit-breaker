package breaker

import "github.com/shuklasaharsh/circuitbreaker/storage"

type State = storage.State

const (
	StateClosed   = storage.StateClosed
	StateOpen     = storage.StateOpen
	StateHalfOpen = storage.StateHalfOpen
)
