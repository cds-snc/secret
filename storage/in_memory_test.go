package storage

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDelete(t *testing.T) {
	t.Parallel()

	var backend StorageBackend = &InMemoryStorageBackend{}
	backend.Init(map[string]string{})
	uuid, _ := backend.Store([]byte("data"), []byte("key"), 0)
	err := backend.Delete(uuid)
	if err != nil {
		t.Error("Expected nil, got", err)
	}
	_, _, err = backend.Retrieve(uuid)
	if err == nil {
		t.Error("Expected error, got nil")
	}

}

func TestInit(t *testing.T) {
	t.Parallel()

	var backend StorageBackend = &InMemoryStorageBackend{}
	err := backend.Init(map[string]string{})
	if err != nil {
		t.Error("Expected nil, got", err)
	}
}

func TestPurge(t *testing.T) {
	t.Parallel()

	var backend StorageBackend = &InMemoryStorageBackend{}
	backend.Init(map[string]string{})
	backend.Store([]byte("data"), []byte("key"), time.Now().Add(-time.Hour).Unix())
	backend.(*InMemoryStorageBackend).purge()
	if backend.(*InMemoryStorageBackend).size() != 0 {
		t.Error("Expected 0, got", backend.(*InMemoryStorageBackend).size())
	}
}

func TestStore(t *testing.T) {
	t.Parallel()

	var backend StorageBackend = &InMemoryStorageBackend{}
	backend.Init(map[string]string{})
	uuid, err := backend.Store([]byte("data"), []byte("key"), 0)
	if err != nil {
		t.Error("Expected nil, got", err)
	}
	if uuid.String() == "" {
		t.Error("Expected UUID, got empty string")
	}
}

func TestRetrieveWithTTLInFuture(t *testing.T) {
	t.Parallel()

	var backend StorageBackend = &InMemoryStorageBackend{}
	backend.Init(map[string]string{})
	uuid, _ := backend.Store([]byte("data"), []byte("key"), time.Now().Add(time.Hour).Unix())
	data, key, err := backend.Retrieve(uuid)
	if err != nil {
		t.Error("Expected nil, got", err)
	}
	if string(data) != "data" {
		t.Error("Expected data, got", string(data))
	}
	if string(key) != "key" {
		t.Error("Expected key, got", string(key))
	}
}

func TestRetrieveWithTTLInPast(t *testing.T) {
	t.Parallel()

	var backend StorageBackend = &InMemoryStorageBackend{}
	backend.Init(map[string]string{})
	uuid, _ := backend.Store([]byte("data"), []byte("key"), time.Now().Add(-time.Hour).Unix())
	_, _, err := backend.Retrieve(uuid)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestRetrieveNotFound(t *testing.T) {
	t.Parallel()

	var backend StorageBackend = &InMemoryStorageBackend{}
	backend.Init(map[string]string{})
	_, _, err := backend.Retrieve(uuid.Nil)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}
