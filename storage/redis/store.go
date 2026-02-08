package redis

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/shuklasaharsh/circuitbreaker/storage"
)

// Client must return storage.ErrNotFound for missing keys and storage.ErrConflict on watch conflicts.
type Client interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Watch(ctx context.Context, key string, fn func(Tx) error) error
}

type Tx interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
}

type Store struct {
	client     Client
	keyPrefix  string
	ttl        time.Duration
	codec      storage.Codec
	maxRetries int
}

type Option func(*Store)

func WithKeyPrefix(prefix string) Option {
	return func(s *Store) {
		s.keyPrefix = prefix
	}
}

func WithTTL(ttl time.Duration) Option {
	return func(s *Store) {
		s.ttl = ttl
	}
}

func WithCodec(codec storage.Codec) Option {
	return func(s *Store) {
		if codec != nil {
			s.codec = codec
		}
	}
}

func WithMaxRetries(n int) Option {
	return func(s *Store) {
		if n < 0 {
			n = 0
		}
		s.maxRetries = n
	}
}

func New(client Client, opts ...Option) (*Store, error) {
	if client == nil {
		return nil, stderrors.New("redis client cannot be nil")
	}

	store := &Store{
		client:     client,
		keyPrefix:  "",
		ttl:        0,
		codec:      storage.JSONCodec{},
		maxRetries: 3,
	}

	for _, opt := range opts {
		opt(store)
	}

	if store.codec == nil {
		store.codec = storage.JSONCodec{}
	}

	return store, nil
}

func (s *Store) Load(ctx context.Context, name string) (storage.Record, error) {
	value, err := s.client.Get(ctx, s.key(name))
	if err != nil {
		if stderrors.Is(err, storage.ErrNotFound) {
			return storage.Record{}, storage.ErrNotFound
		}
		return storage.Record{}, err
	}
	return s.codec.Unmarshal([]byte(value))
}

func (s *Store) Save(ctx context.Context, name string, record storage.Record) error {
	payload, err := s.codec.Marshal(record)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, s.key(name), string(payload), s.ttl)
}

func (s *Store) Update(ctx context.Context, name string, fn func(storage.Record) (storage.Record, error)) (storage.Record, error) {
	var updated storage.Record
	var lastErr error

	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		err := s.client.Watch(ctx, s.key(name), func(tx Tx) error {
			record, err := s.loadFromTx(ctx, tx, name)
			if err != nil {
				return err
			}
			updated, err = fn(record)
			if err != nil {
				return err
			}
			payload, err := s.codec.Marshal(updated)
			if err != nil {
				return err
			}
			return tx.Set(ctx, s.key(name), string(payload), s.ttl)
		})
		if err == nil {
			return updated, nil
		}
		if stderrors.Is(err, storage.ErrConflict) {
			lastErr = err
			continue
		}
		return storage.Record{}, err
	}

	if lastErr != nil {
		return storage.Record{}, lastErr
	}
	return storage.Record{}, storage.ErrConflict
}

func (s *Store) loadFromTx(ctx context.Context, tx Tx, name string) (storage.Record, error) {
	value, err := tx.Get(ctx, s.key(name))
	if err != nil {
		if stderrors.Is(err, storage.ErrNotFound) {
			return storage.DefaultRecord(), nil
		}
		return storage.Record{}, err
	}
	record, err := s.codec.Unmarshal([]byte(value))
	if err != nil {
		if stderrors.Is(err, storage.ErrNotFound) {
			return storage.DefaultRecord(), nil
		}
		return storage.Record{}, err
	}
	return record, nil
}

func (s *Store) key(name string) string {
	if s.keyPrefix == "" {
		return name
	}
	return s.keyPrefix + ":" + name
}
