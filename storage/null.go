package storage

import (
	"time"

	"github.com/google/uuid"
)

// NullBackend is a storage backend that does nothing

type NullBackend struct{}

func (b *NullBackend) Claim(id uuid.UUID, lease time.Duration) (ClaimedSecret, error) {
	return ClaimedSecret{}, ErrSecretUnavailable
}

func (b *NullBackend) Consume(id, token uuid.UUID) error {
	return nil
}

func (b *NullBackend) Delete(id uuid.UUID) error {
	return nil
}

func (b *NullBackend) Init(map[string]string) error {
	return nil
}

func (b *NullBackend) Store(data, key []byte, ttl int64) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (b *NullBackend) Release(id, token uuid.UUID) error {
	return nil
}
