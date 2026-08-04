package storage

import (
	"errors"
	"testing"
	"time"

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

func TestNullBackendClaim(t *testing.T) {
	t.Parallel()

	backend := NullBackend{}
	id := uuid.New()
	_, err := backend.Claim(id, time.Minute)
	if !errors.Is(err, ErrSecretUnavailable) {
		t.Errorf("Claim() = %v, want ErrSecretUnavailable", err)
	}
}

func TestNullBackendClaimLifecycle(t *testing.T) {
	t.Parallel()

	backend := NullBackend{}
	id := uuid.New()
	token := uuid.New()
	if err := backend.Consume(id, token); err != nil {
		t.Errorf("Consume() failed: %v", err)
	}
	if err := backend.Release(id, token); err != nil {
		t.Errorf("Release() failed: %v", err)
	}
}
