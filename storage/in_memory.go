package storage

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type pair struct {
	Data       []byte
	Key        []byte
	TTL        int64
	ClaimToken uuid.UUID
	ClaimUntil time.Time
}

// InMemoryStorageBackend is a storage backend that stores data in memory
type InMemoryStorageBackend struct {
	m    sync.Mutex
	data map[uuid.UUID]pair
}

// Delete deletes data from the storage backend
func (b *InMemoryStorageBackend) Delete(id uuid.UUID) error {
	b.m.Lock()
	defer b.m.Unlock()

	delete(b.data, id)
	return nil
}

// Init initializes the storage backend
func (b *InMemoryStorageBackend) Init(map[string]string) error {
	b.data = make(map[uuid.UUID]pair)
	b.m = sync.Mutex{}

	// Purge data that is expired from the storage backend every minute
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			b.purge()
		}
	}()

	return nil
}

// Purge data that is expired from the storage backend
// The function itself may not be as efficient at it is O(n)
// however, Go is fast enough to purge 100k+ entries in a few ms
func (b *InMemoryStorageBackend) purge() error {
	b.m.Lock()
	defer b.m.Unlock()

	for id, pair := range b.data {
		if pair.TTL < time.Now().Unix() {
			delete(b.data, id)
		}
	}
	return nil
}

func (b *InMemoryStorageBackend) size() int {
	b.m.Lock()
	defer b.m.Unlock()

	return len(b.data)
}

// Store stores data in the storage backend
func (b *InMemoryStorageBackend) Store(data, key []byte, TTL int64) (uuid.UUID, error) {
	b.m.Lock()
	defer b.m.Unlock()

	id := uuid.New()
	b.data[id] = pair{Data: data, Key: key, TTL: TTL}
	return id, nil
}

// Claim atomically reserves a secret for a single consumer for the lease duration.
func (b *InMemoryStorageBackend) Claim(id uuid.UUID, lease time.Duration) (ClaimedSecret, error) {
	if lease <= 0 {
		return ClaimedSecret{}, fmt.Errorf("claim lease must be positive")
	}

	b.m.Lock()
	defer b.m.Unlock()

	secret, ok := b.data[id]
	if !ok {
		return ClaimedSecret{}, ErrSecretUnavailable
	}

	now := time.Now()
	if secret.TTL < now.Unix() {
		delete(b.data, id)
		return ClaimedSecret{}, ErrSecretUnavailable
	}

	if secret.ClaimToken != uuid.Nil && secret.ClaimUntil.After(now) {
		return ClaimedSecret{}, ErrSecretUnavailable
	}

	token := uuid.New()
	secret.ClaimToken = token
	secret.ClaimUntil = now.Add(lease)
	b.data[id] = secret

	return ClaimedSecret{
		Data:  append([]byte(nil), secret.Data...),
		Key:   append([]byte(nil), secret.Key...),
		Token: token,
	}, nil
}

// Consume permanently deletes a secret if the caller still owns its claim.
func (b *InMemoryStorageBackend) Consume(id, token uuid.UUID) error {
	b.m.Lock()
	defer b.m.Unlock()

	secret, ok := b.data[id]
	if !ok || secret.ClaimToken != token || !secret.ClaimUntil.After(time.Now()) {
		return ErrClaimLost
	}

	delete(b.data, id)
	return nil
}

// Release makes a claimed secret immediately available for another attempt.
func (b *InMemoryStorageBackend) Release(id, token uuid.UUID) error {
	b.m.Lock()
	defer b.m.Unlock()

	secret, ok := b.data[id]
	if !ok || secret.ClaimToken != token {
		return ErrClaimLost
	}

	secret.ClaimToken = uuid.Nil
	secret.ClaimUntil = time.Time{}
	b.data[id] = secret
	return nil
}
