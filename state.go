package breaker

type State uint

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	return []string{"Closed", "Open", "Half Open"}[s]
}
