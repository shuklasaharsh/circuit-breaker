package storage

import (
	"errors"
	"testing"
	"time"
)

func TestStateString(t *testing.T) {
	cases := []struct {
		state    State
		expected string
	}{
		{StateClosed, "Closed"},
		{StateOpen, "Open"},
		{StateHalfOpen, "Half Open"},
		{State(99), "Unknown"},
	}

	for _, tc := range cases {
		if tc.state.String() != tc.expected {
			t.Fatalf("expected %q, got %q", tc.expected, tc.state.String())
		}
	}
}

func TestDefaultRecord(t *testing.T) {
	record := DefaultRecord()
	if record.State != StateClosed {
		t.Fatalf("expected closed, got %v", record.State)
	}
	if record.Failures != 0 || record.Successes != 0 {
		t.Fatalf("expected zero counts, got failures=%d successes=%d", record.Failures, record.Successes)
	}
}

func TestJSONCodecRoundTrip(t *testing.T) {
	codec := JSONCodec{}
	record := Record{
		State:           StateOpen,
		Failures:        2,
		Successes:       1,
		LastFailureTime: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	data, err := codec.Marshal(record)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	decoded, err := codec.Unmarshal(data)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.State != record.State || decoded.Failures != record.Failures || decoded.Successes != record.Successes {
		t.Fatalf("decoded mismatch: %#v", decoded)
	}
	if !decoded.LastFailureTime.Equal(record.LastFailureTime) {
		t.Fatalf("expected %v, got %v", record.LastFailureTime, decoded.LastFailureTime)
	}
}

func TestJSONCodecEmptyData(t *testing.T) {
	codec := JSONCodec{}
	_, err := codec.Unmarshal(nil)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestJSONCodecInvalidData(t *testing.T) {
	codec := JSONCodec{}
	_, err := codec.Unmarshal([]byte("not-json"))
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("expected non-ErrNotFound error, got %v", err)
	}
}
