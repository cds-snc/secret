package main

import (
	"log"

	app "github.com/cds-snc/secret"
	"github.com/cds-snc/secret/encryption"
	"github.com/cds-snc/secret/storage"
)

func main() {
	encryption := &encryption.RsaKeyPair{}
	config := map[string]string{
		"publicKeyPath":  "keys/public.pem",
		"privateKeyPath": "keys/private.pem",
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
