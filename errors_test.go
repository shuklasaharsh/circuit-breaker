package breaker

import (
	"strings"
	"testing"
)

func TestErrorMessages(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		substrs []string
	}{
		{"invalid-threshold", ErrInvalidThresholdValue, []string{"threshold", "code"}},
		{"invalid-duration", ErrInvalidDuration, []string{"duration", "code"}},
		{"nil-function", ErrNilFunction, []string{"function", "code"}},
		{"circuit-open", ErrCircuitOpen, []string{"circuit breaker is open", "code"}},
	}

	for _, tc := range cases {
		if tc.err == nil {
			t.Fatalf("%s: expected error", tc.name)
		}
		msg := tc.err.Error()
		for _, substr := range tc.substrs {
			if !strings.Contains(msg, substr) {
				t.Fatalf("%s: expected %q in %q", tc.name, substr, msg)
			}
		}
	}
}
