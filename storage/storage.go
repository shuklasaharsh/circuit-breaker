package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

type State uint8

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "Closed"
	case StateOpen:
		return "Open"
	case StateHalfOpen:
		return "Half Open"
	default:
		return "Unknown"
	}
}

type Record struct {
	State           State     `json:"state"`
	Failures        int64     `json:"failures"`
	Successes       int64     `json:"successes"`
	LastFailureTime time.Time `json:"last_failure_time"`
}

func DefaultRecord() Record {
	return Record{
		State:     StateClosed,
		Failures:  0,
		Successes: 0,
	}
}

var (
	ErrNotFound = errors.New("storage record not found")
	ErrConflict = errors.New("storage record update conflict")
)

type Store interface {
	Load(ctx context.Context, name string) (Record, error)
	Save(ctx context.Context, name string, record Record) error
	Update(ctx context.Context, name string, fn func(Record) (Record, error)) (Record, error)
}

type Codec interface {
	Marshal(record Record) ([]byte, error)
	Unmarshal(data []byte) (Record, error)
}

type JSONCodec struct{}

func (JSONCodec) Marshal(record Record) ([]byte, error) {
	return json.Marshal(record)
}

func (JSONCodec) Unmarshal(data []byte) (Record, error) {
	var record Record
	if len(data) == 0 {
		return record, ErrNotFound
	}
	if err := json.Unmarshal(data, &record); err != nil {
		return Record{}, err
	}
	return record, nil
}
