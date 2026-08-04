package storage

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrSecretUnavailable = errors.New("secret not found, expired, or already claimed")
	ErrClaimLost         = errors.New("secret claim is no longer valid")
)

type ClaimedSecret struct {
	Data  []byte
	Key   []byte
	Token uuid.UUID
}

type StorageBackend interface {
	Claim(id uuid.UUID, lease time.Duration) (ClaimedSecret, error)
	Consume(id, token uuid.UUID) error
	Delete(id uuid.UUID) error
	Init(map[string]string) error
	Release(id, token uuid.UUID) error
	Store(data, key []byte, ttl int64) (uuid.UUID, error)
}
