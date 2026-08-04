package storage

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

const testClaimLease = time.Minute

func newInMemoryBackend(t *testing.T) *InMemoryStorageBackend {
	t.Helper()

	backend := &InMemoryStorageBackend{}
	if err := backend.Init(map[string]string{}); err != nil {
		t.Fatalf("InMemoryStorageBackend.Init() failed: %v", err)
	}
	return backend
}

func TestDelete(t *testing.T) {
	t.Parallel()

	backend := newInMemoryBackend(t)
	id, _ := backend.Store([]byte("data"), []byte("key"), time.Now().Add(time.Hour).Unix(), false)
	if err := backend.Delete(id); err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	if _, err := backend.Claim(id, testClaimLease); !errors.Is(err, ErrSecretUnavailable) {
		t.Fatalf("Claim() after Delete() = %v, want ErrSecretUnavailable", err)
	}
}

func TestInit(t *testing.T) {
	t.Parallel()

	backend := &InMemoryStorageBackend{}
	if err := backend.Init(map[string]string{}); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}
}

func TestPurge(t *testing.T) {
	t.Parallel()

	backend := newInMemoryBackend(t)
	_, _ = backend.Store([]byte("data"), []byte("key"), time.Now().Add(-time.Hour).Unix(), false)
	if err := backend.purge(); err != nil {
		t.Fatalf("purge() failed: %v", err)
	}
	if backend.size() != 0 {
		t.Fatalf("size() = %d, want 0", backend.size())
	}
}

func TestStore(t *testing.T) {
	t.Parallel()

	backend := newInMemoryBackend(t)
	id, err := backend.Store([]byte("data"), []byte("key"), time.Now().Add(time.Hour).Unix(), false)
	if err != nil {
		t.Fatalf("Store() failed: %v", err)
	}
	if id == uuid.Nil {
		t.Fatal("Store() returned a nil UUID")
	}
}

func TestClaimAndConsume(t *testing.T) {
	t.Parallel()

	backend := newInMemoryBackend(t)
	id, _ := backend.Store([]byte("data"), []byte("key"), time.Now().Add(time.Hour).Unix(), true)

	claim, err := backend.Claim(id, testClaimLease)
	if err != nil {
		t.Fatalf("Claim() failed: %v", err)
	}
	if string(claim.Data) != "data" || string(claim.Key) != "key" || claim.Token == uuid.Nil {
		t.Fatalf("Claim() returned an invalid claim: %#v", claim)
	}
	if claim.ClientEncrypted == nil || !*claim.ClientEncrypted {
		t.Fatalf("Claim() did not preserve client encryption metadata: %#v", claim)
	}

	if err := backend.Consume(id, claim.Token); err != nil {
		t.Fatalf("Consume() failed: %v", err)
	}
	if _, err := backend.Claim(id, testClaimLease); !errors.Is(err, ErrSecretUnavailable) {
		t.Fatalf("Claim() after Consume() = %v, want ErrSecretUnavailable", err)
	}
}

func TestClaimExpiredOrMissingSecret(t *testing.T) {
	t.Parallel()

	backend := newInMemoryBackend(t)
	id, _ := backend.Store([]byte("data"), []byte("key"), time.Now().Add(-time.Hour).Unix(), false)

	for _, testID := range []uuid.UUID{id, uuid.New()} {
		if _, err := backend.Claim(testID, testClaimLease); !errors.Is(err, ErrSecretUnavailable) {
			t.Fatalf("Claim(%s) = %v, want ErrSecretUnavailable", testID, err)
		}
	}
}

func TestConcurrentClaimsHaveExactlyOneWinner(t *testing.T) {
	t.Parallel()

	backend := newInMemoryBackend(t)
	id, _ := backend.Store([]byte("data"), []byte("key"), time.Now().Add(time.Hour).Unix(), false)

	const contenders = 32
	start := make(chan struct{})
	var wg sync.WaitGroup
	var winners atomic.Int32
	winnerTokens := make(chan uuid.UUID, contenders)

	for range contenders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			claim, err := backend.Claim(id, testClaimLease)
			if err == nil {
				winners.Add(1)
				winnerTokens <- claim.Token
				return
			}
			if !errors.Is(err, ErrSecretUnavailable) {
				t.Errorf("Claim() returned unexpected error: %v", err)
			}
		}()
	}

	close(start)
	wg.Wait()
	close(winnerTokens)

	if winners.Load() != 1 {
		t.Fatalf("successful claims = %d, want exactly 1", winners.Load())
	}
	if err := backend.Consume(id, <-winnerTokens); err != nil {
		t.Fatalf("Consume() winning claim failed: %v", err)
	}
}

func TestReleaseAllowsImmediateRetry(t *testing.T) {
	t.Parallel()

	backend := newInMemoryBackend(t)
	id, _ := backend.Store([]byte("data"), []byte("key"), time.Now().Add(time.Hour).Unix(), false)

	first, err := backend.Claim(id, testClaimLease)
	if err != nil {
		t.Fatalf("first Claim() failed: %v", err)
	}
	if err := backend.Release(id, first.Token); err != nil {
		t.Fatalf("Release() failed: %v", err)
	}

	second, err := backend.Claim(id, testClaimLease)
	if err != nil {
		t.Fatalf("Claim() after Release() failed: %v", err)
	}
	if second.Token == first.Token {
		t.Fatal("Claim() after Release() reused the previous token")
	}
}

func TestExpiredClaimCanBeReclaimed(t *testing.T) {
	t.Parallel()

	backend := newInMemoryBackend(t)
	id, _ := backend.Store([]byte("data"), []byte("key"), time.Now().Add(time.Hour).Unix(), false)

	first, err := backend.Claim(id, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("first Claim() failed: %v", err)
	}
	time.Sleep(20 * time.Millisecond)

	second, err := backend.Claim(id, testClaimLease)
	if err != nil {
		t.Fatalf("Claim() after lease expiry failed: %v", err)
	}
	if second.Token == first.Token {
		t.Fatal("reclaimed secret reused the expired token")
	}
	if err := backend.Consume(id, first.Token); !errors.Is(err, ErrClaimLost) {
		t.Fatalf("Consume() with expired token = %v, want ErrClaimLost", err)
	}
}
