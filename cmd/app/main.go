package main

import (
	"log"
	"os"

	app "github.com/cds-snc/secret"
	"github.com/cds-snc/secret/encryption"
	"github.com/cds-snc/secret/storage"
)

func main() {
	encryption := &encryption.RsaKeyPair{}
	config := map[string]string{
		"publicKeyPath":  envOrDefault("PUBLIC_KEY_PATH", "keys/public.pem"),
		"privateKeyPath": envOrDefault("PRIVATE_KEY_PATH", "keys/private.pem"),
	}
	err := encryption.Init(config)
	if err != nil {
		log.Fatal(err)
	}

	storage := &storage.InMemoryStorageBackend{}
	err = storage.Init(map[string]string{})
	if err != nil {
		log.Fatal(err)
	}

	app := app.CreateApp(encryption, storage)
	log.Fatal(app.Listen(":3000"))
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
