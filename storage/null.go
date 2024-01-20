package storage

import (
	"github.com/google/uuid"
)

// NullBackend is a storage backend that does nothing

type NullBackend struct{}

func (b *NullBackend) Delete(id uuid.UUID) error {
	return nil
}

func (b *NullBackend) Init(map[string]string) error {
	return nil
}

func (b *NullBackend) Store(data, key []byte, ttl int64) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (b *NullBackend) Retrieve(id uuid.UUID) ([]byte, []byte, error) {
	return nil, nil, nil
}
