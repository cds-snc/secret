package storage

import "github.com/google/uuid"

type StorageBackend interface {
	Delete(id uuid.UUID) error
	Init(map[string]string) error
	Store(data, key []byte, ttl int64) (uuid.UUID, error)
	Retrieve(id uuid.UUID) ([]byte, []byte, error)
}
