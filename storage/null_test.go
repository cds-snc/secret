package storage

import (
	"testing"

	"github.com/google/uuid"
)

func TestNullBackendDelete(t *testing.T) {
	t.Parallel()

	backend := NullBackend{}
	err := backend.Delete(uuid.New())
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestNullBackendInit(t *testing.T) {
	t.Parallel()

	backend := NullBackend{}
	err := backend.Init(map[string]string{})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestNullBackendStore(t *testing.T) {
	t.Parallel()

	backend := NullBackend{}
	data := []byte("hello world")
	key := []byte("key")
	_, err := backend.Store(data, key, 0)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestNullBackendRetrieve(t *testing.T) {
	t.Parallel()

	backend := NullBackend{}
	id := uuid.New()
	_, _, err := backend.Retrieve(id)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
