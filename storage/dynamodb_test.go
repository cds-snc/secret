package storage

import (
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMain(m *testing.M) {
	// Ensure that a DynamoDB table exists
	backend := DynamoDBBackend{}

	_ = backend.Init(map[string]string{
		"endpoint":   getDynamoDBHost,
		"region":     "ca-central-1",
		"table_name": "secrets",
	})

	_ = backend.createTable()

	// setup
	code := m.Run()
	// teardown
	os.Exit(code)
}

var getDynamoDBHost string = func() string {
	host := "http://dynamodb-local:8000"

	if h := os.Getenv("DYNAMODB_HOST"); h != "" {
		host = h
	}

	return host
}()

func TestDynamoDBBackendDelete(t *testing.T) {
	t.Parallel()

	backend := DynamoDBBackend{}

	_ = backend.Init(map[string]string{
		"endpoint":   getDynamoDBHost,
		"region":     "ca-central-1",
		"table_name": "secrets",
	})

	err := backend.Delete(uuid.New())

	if err != nil {
		t.Errorf("DynamoDBBackend.Delete() failed: %s", err)
	}
}

func TestDynamoDBBackendInit(t *testing.T) {
	t.Parallel()

	backend := DynamoDBBackend{}

	err := backend.Init(map[string]string{
		"region":     "ca-central-1",
		"table_name": "secrets",
	})

	if err != nil {
		t.Errorf("DynamoDBBackend.Init() failed: %s", err)
	}
}

func TestDynamoDBBackendInitMissingRegion(t *testing.T) {
	t.Parallel()

	backend := DynamoDBBackend{}

	err := backend.Init(map[string]string{
		"table_name": "secrets",
	})

	if err == nil {
		t.Errorf("DynamoDBBackend.Init() should fail without region")
	}
}

func TestDynamoDBBackendInitMissingTableName(t *testing.T) {
	t.Parallel()

	backend := DynamoDBBackend{}

	err := backend.Init(map[string]string{
		"region": "ca-central-1",
	})

	if err == nil {
		t.Errorf("DynamoDBBackend.Init() should fail without table_name")
	}
}

func TestDynamoDBBackendInitWithEndpoint(t *testing.T) {
	t.Parallel()

	backend := DynamoDBBackend{}

	err := backend.Init(map[string]string{
		"region":     "ca-central-1",
		"table_name": "secrets",
		"endpoint":   getDynamoDBHost,
	})

	if err != nil {
		t.Errorf("DynamoDBBackend.Init() failed: %s", err)
	}
}

func TestDynamoDBBackendStore(t *testing.T) {
	t.Parallel()

	backend := DynamoDBBackend{}

	_ = backend.Init(map[string]string{
		"endpoint":   getDynamoDBHost,
		"region":     "ca-central-1",
		"table_name": "secrets",
	})

	id, err := backend.Store([]byte("test"), []byte("test"), 1000, false)

	if err != nil {
		t.Errorf("DynamoDBBackend.Store() failed: %s", err)
	}

	if id == uuid.Nil {
		t.Errorf("DynamoDBBackend.Store() returned a nil UUID")
	}
}

func TestDynamoDBBackendClaimAndConsume(t *testing.T) {
	t.Parallel()

	backend := DynamoDBBackend{}

	_ = backend.Init(map[string]string{
		"endpoint":   getDynamoDBHost,
		"region":     "ca-central-1",
		"table_name": "secrets",
	})

	id, err := backend.Store([]byte("test"), []byte("key"), time.Now().Add(time.Hour).Unix(), true)

	if err != nil {
		t.Errorf("DynamoDBBackend.Store() failed: %s", err)
	}

	claim, err := backend.Claim(id, time.Minute)

	if err != nil {
		t.Fatalf("DynamoDBBackend.Claim() failed: %s", err)
	}

	if string(claim.Data) != "test" {
		t.Errorf("DynamoDBBackend.Claim() returned the wrong data")
	}

	if string(claim.Key) != "key" {
		t.Errorf("DynamoDBBackend.Claim() returned the wrong key")
	}
	if claim.ClientEncrypted == nil || !*claim.ClientEncrypted {
		t.Errorf("DynamoDBBackend.Claim() did not preserve client encryption metadata")
	}

	if err := backend.Consume(id, claim.Token); err != nil {
		t.Fatalf("DynamoDBBackend.Consume() failed: %s", err)
	}
	if _, err := backend.Claim(id, time.Minute); !errors.Is(err, ErrSecretNotFound) {
		t.Errorf("Claim() after Consume() = %v, want ErrSecretNotFound", err)
	}
}

func TestDynamoDBBackendClaimMissingSecret(t *testing.T) {
	t.Parallel()

	backend := DynamoDBBackend{}
	_ = backend.Init(map[string]string{
		"endpoint":   getDynamoDBHost,
		"region":     "ca-central-1",
		"table_name": "secrets",
	})

	if _, err := backend.Claim(uuid.New(), time.Minute); !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("Claim() missing secret = %v, want ErrSecretNotFound", err)
	}
}

func TestDynamoDBBackendClaimWithTTLInPast(t *testing.T) {
	t.Parallel()

	backend := DynamoDBBackend{}

	_ = backend.Init(map[string]string{
		"endpoint":   getDynamoDBHost,
		"region":     "ca-central-1",
		"table_name": "secrets",
	})

	id, err := backend.Store([]byte("test"), []byte("key"), time.Now().Add(-time.Hour).Unix(), false)

	if err != nil {
		t.Errorf("DynamoDBBackend.Store() failed: %s", err)
	}

	_, err = backend.Claim(id, time.Minute)

	if !errors.Is(err, ErrSecretUnavailable) {
		t.Errorf("DynamoDBBackend.Claim() = %v, want ErrSecretUnavailable", err)
	}
}

func TestDynamoDBBackendConcurrentClaimsHaveExactlyOneWinner(t *testing.T) {
	t.Parallel()

	backend := DynamoDBBackend{}
	_ = backend.Init(map[string]string{
		"endpoint":   getDynamoDBHost,
		"region":     "ca-central-1",
		"table_name": "secrets",
	})
	id, err := backend.Store([]byte("test"), []byte("key"), time.Now().Add(time.Hour).Unix(), false)
	if err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	const contenders = 16
	start := make(chan struct{})
	var wg sync.WaitGroup
	var winners atomic.Int32
	tokens := make(chan uuid.UUID, contenders)

	for range contenders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			claim, claimErr := backend.Claim(id, time.Minute)
			if claimErr == nil {
				winners.Add(1)
				tokens <- claim.Token
				return
			}
			if !errors.Is(claimErr, ErrSecretUnavailable) {
				t.Errorf("Claim() returned unexpected error: %v", claimErr)
			}
		}()
	}

	close(start)
	wg.Wait()
	close(tokens)

	if winners.Load() != 1 {
		t.Fatalf("successful claims = %d, want exactly 1", winners.Load())
	}
	if err := backend.Consume(id, <-tokens); err != nil {
		t.Fatalf("Consume() winning claim failed: %v", err)
	}
}

func TestDynamoDBBackendReleaseAllowsRetry(t *testing.T) {
	t.Parallel()

	backend := DynamoDBBackend{}
	_ = backend.Init(map[string]string{
		"endpoint":   getDynamoDBHost,
		"region":     "ca-central-1",
		"table_name": "secrets",
	})
	id, err := backend.Store([]byte("test"), []byte("key"), time.Now().Add(time.Hour).Unix(), false)
	if err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	first, err := backend.Claim(id, time.Minute)
	if err != nil {
		t.Fatalf("first Claim() failed: %v", err)
	}
	if err := backend.Release(id, first.Token); err != nil {
		t.Fatalf("Release() failed: %v", err)
	}
	second, err := backend.Claim(id, time.Minute)
	if err != nil {
		t.Fatalf("Claim() after Release() failed: %v", err)
	}
	if second.Token == first.Token {
		t.Fatal("Claim() after Release() reused the old token")
	}
}
