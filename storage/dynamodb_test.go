package storage

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMain(m *testing.M) {
	// Ensure that a DynamoDB table exists
	backend := DynamoDBBackend{}

	_ = backend.Init(map[string]string{
		"endpoint":   getDynamoDBHost(),
		"region":     "ca-central-1",
		"table_name": "secrets",
	})

	_ = backend.createTable()

	// setup
	code := m.Run()
	// teardown
	os.Exit(code)
}

func getDynamoDBHost() string {
	host := "http://dynamodb-local:8000"

	if h := os.Getenv("DYNAMODB_HOST"); h != "" {
		host = h
	}

	return host
}

func TestDynamoDBBackendDelete(t *testing.T) {
	t.Parallel()

	backend := DynamoDBBackend{}

	_ = backend.Init(map[string]string{
		"endpoint":   getDynamoDBHost(),
		"region":     "ca-central-1",
		"table_name": "test",
	})

	err := backend.Delete(uuid.New())

	if err == nil {
		t.Errorf("DynamoDBBackend.Delete() succeeded and should have fialed with a non-existent ID")
	}
}

func TestDynamoDBBackendInit(t *testing.T) {
	t.Parallel()

	backend := DynamoDBBackend{}

	err := backend.Init(map[string]string{
		"region":     "ca-central-1",
		"table_name": "test",
	})

	if err != nil {
		t.Errorf("DynamoDBBackend.Init() failed: %s", err)
	}
}

func TestDynamoDBBackendInitMissingRegion(t *testing.T) {
	t.Parallel()

	backend := DynamoDBBackend{}

	err := backend.Init(map[string]string{
		"table_name": "test",
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
		"table_name": "test",
		"endpoint":   getDynamoDBHost(),
	})

	if err != nil {
		t.Errorf("DynamoDBBackend.Init() failed: %s", err)
	}
}

func TestDynamoDBBackendStore(t *testing.T) {
	t.Parallel()

	backend := DynamoDBBackend{}

	_ = backend.Init(map[string]string{
		"endpoint":   getDynamoDBHost(),
		"region":     "ca-central-1",
		"table_name": "secrets",
	})

	id, err := backend.Store([]byte("test"), []byte("test"), 1000)

	if err != nil {
		t.Errorf("DynamoDBBackend.Store() failed: %s", err)
	}

	if id == uuid.Nil {
		t.Errorf("DynamoDBBackend.Store() returned a nil UUID")
	}
}

func TestDynamoDBBackendRetrieveWithTTLInFuture(t *testing.T) {
	t.Parallel()

	backend := DynamoDBBackend{}

	_ = backend.Init(map[string]string{
		"endpoint":   getDynamoDBHost(),
		"region":     "ca-central-1",
		"table_name": "secrets",
	})

	id, err := backend.Store([]byte("test"), []byte("key"), time.Now().Add(time.Hour).Unix())

	if err != nil {
		t.Errorf("DynamoDBBackend.Store() failed: %s", err)
	}

	data, key, err := backend.Retrieve(id)

	if err != nil {
		t.Errorf("DynamoDBBackend.Retrieve() failed: %s", err)
	}

	if string(data) != "test" {
		t.Errorf("DynamoDBBackend.Retrieve() returned the wrong data")
	}

	if string(key) != "key" {
		t.Errorf("DynamoDBBackend.Retrieve() returned the wrong key")
	}
}

func TestDynamoDBBackendRetrieveWithTTLInPast(t *testing.T) {
	t.Parallel()

	backend := DynamoDBBackend{}

	_ = backend.Init(map[string]string{
		"endpoint":   getDynamoDBHost(),
		"region":     "ca-central-1",
		"table_name": "secrets",
	})

	id, err := backend.Store([]byte("test"), []byte("key"), time.Now().Add(-time.Hour).Unix())

	if err != nil {
		t.Errorf("DynamoDBBackend.Store() failed: %s", err)
	}

	_, _, err = backend.Retrieve(id)

	if err == nil {
		t.Errorf("DynamoDBBackend.Retrieve() should have failed")
	}
}
