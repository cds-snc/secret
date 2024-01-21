package storage

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type pair struct {
	Data []byte
	Key  []byte
	TTL  int64
}

// InMemoryStorageBackend is a storage backend that stores data in memory
type InMemoryStorageBackend struct {
	data map[uuid.UUID]pair
}

// Delete deletes data from the storage backend
func (b *InMemoryStorageBackend) Delete(id uuid.UUID) error {
	delete(b.data, id)
	return nil
}

// Init initializes the storage backend
func (b *InMemoryStorageBackend) Init(map[string]string) error {
	b.data = make(map[uuid.UUID]pair)

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
	for id, pair := range b.data {
		if pair.TTL < time.Now().Unix() {
			b.Delete(id)
		}
	}
	return nil
}

func (b *InMemoryStorageBackend) size() int {
	return len(b.data)
}

// Store stores data in the storage backend
func (b *InMemoryStorageBackend) Store(data, key []byte, TTL int64) (uuid.UUID, error) {
	uuid := uuid.New()
	b.data[uuid] = pair{Data: data, Key: key, TTL: TTL}
	return uuid, nil
}

// Retrieve retrieves data from the storage backend
func (b *InMemoryStorageBackend) Retrieve(id uuid.UUID) ([]byte, []byte, error) {
	if _, ok := b.data[id]; ok {

		// Check if TTL timestamp is less than current unix time
		if b.data[id].TTL < time.Now().Unix() {
			b.Delete(id)
			return nil, nil, fmt.Errorf("UUID not found")
		}

		return b.data[id].Data, b.data[id].Key, nil
	}

	return nil, nil, fmt.Errorf("UUID not found")
}
