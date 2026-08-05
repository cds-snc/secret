package app

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/template/html/v2"
	"github.com/google/uuid"

	"github.com/cds-snc/secret/encryption"
	"github.com/cds-snc/secret/storage"
)

const MAX_AGE_IN_DAYS = 7
const MAX_SECRET_LENGTH = 64_000
const CLIENT_ENCRYPTION_PREFIX = "emc:v2:"
const CLIENT_ENCRYPTION_ITERATIONS = 600_000

// Keep the claim lease longer than the deployed Lambda's 60-second timeout so
// a request cannot lose its claim while it is still able to return a response.
const SECRET_CLAIM_LEASE = 90 * time.Second

var (
	errInvalidTTL                      = errors.New("invalid TTL")
	errSecretTooLong                   = errors.New("secret too long")
	errInvalidClientEncryptionEnvelope = errors.New("invalid client encryption envelope")
)

type AppConfig struct {
	RequireAdditionalPassword bool
}

func AppConfigFromEnv() AppConfig {
	requireAdditionalPassword, _ := strconv.ParseBool(os.Getenv("REQUIRE_ADDITIONAL_PASSWORD"))

	return AppConfig{
		RequireAdditionalPassword: requireAdditionalPassword,
	}
}

func CreateApp(encryptionBackend encryption.EncryptionBackend, storageBackend storage.StorageBackend) *fiber.App {
	return CreateAppWithConfig(encryptionBackend, storageBackend, AppConfigFromEnv())
}

func CreateAppWithConfig(encryptionBackend encryption.EncryptionBackend, storageBackend storage.StorageBackend, config AppConfig) *fiber.App {
	engine := html.New("./views", ".html")

	locales := loadLocales()

	engine.AddFunc("t", func(toTranslate string, lang string) string {
		if locales[lang][toTranslate] != "" {
			return locales[lang][toTranslate]
		}
		return toTranslate
	})

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"Lang":                      "en",
			"OtherLang":                 getOtherLanguage("en"),
			"RequireAdditionalPassword": config.RequireAdditionalPassword,
		}, "base")
	})

	app.Get("/version", func(c *fiber.Ctx) error {
		version := os.Getenv("GIT_SHA")
		if version == "" {
			version = "dev"
		}

		return c.JSON(fiber.Map{
			"version": version,
		})
	})

	app.Get("/.well-known/security.txt", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/plain")
		securityTxt := `Contact: mailto:ZZTBSCYBERS@tbs-sct.gc.ca
Contact: https://hackerone.com/tbs-sct/
Canonical: https://cdssandbox.xyz/.well-known/security.txt
Expires: 2027-04-01T00:00:00Z 
Preferred-Languages: en, fr`
		return c.SendString(securityTxt)
	})

	app.Get("/:language", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"Lang":                      c.Params("language"),
			"OtherLang":                 getOtherLanguage(c.Params("language")),
			"RequireAdditionalPassword": config.RequireAdditionalPassword,
		}, "base")
	})

	app.Get("/:language/view/:id", func(c *fiber.Ctx) error {
		//Convert the id to a UUID
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Warn(err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid UUID")
		}

		return c.Render("view", fiber.Map{
			"Lang":                      c.Params("language"),
			"OtherLang":                 getOtherLanguage(c.Params("language")),
			"ViewId":                    id,
			"RequireAdditionalPassword": config.RequireAdditionalPassword,
		}, "base")
	})

	app.Post("/decrypt/:id", func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderCacheControl, "no-store, max-age=0")

		//Convert the id to a UUID
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Warn(err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid UUID")
		}

		// Atomically reserve the secret so concurrent requests cannot both decrypt it.
		claimedSecret, err := storageBackend.Claim(id, SECRET_CLAIM_LEASE)
		if err != nil {
			log.Warn(err)
			if errors.Is(err, storage.ErrSecretNotFound) || errors.Is(err, storage.ErrSecretUnavailable) {
				return c.Status(fiber.StatusNotFound).SendString("Secret not found")
			}
			return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
		}

		claimActive := true
		defer func() {
			if claimActive {
				if releaseErr := storageBackend.Release(id, claimedSecret.Token); releaseErr != nil {
					log.Error(releaseErr)
				}
			}
		}()

		//Decrypt the data
		decryptedData, err := encryptionBackend.Decrypt(claimedSecret.Data, claimedSecret.Key)
		if err != nil {
			log.Error(err)
			return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
		}

		// Consume only the claim owned by this request before revealing the secret.
		err = storageBackend.Consume(id, claimedSecret.Token)
		if err != nil {
			log.Error(err)
			return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
		}
		claimActive = false

		// Return a JSON response with the decrypted data
		clientEncrypted := isClientEncryptionEnvelope(string(decryptedData))
		if claimedSecret.ClientEncrypted != nil {
			clientEncrypted = *claimedSecret.ClientEncrypted
		}

		return c.JSON(fiber.Map{
			"body":             string(decryptedData),
			"client_encrypted": clientEncrypted,
		})
	})

	app.Delete("/delete/:id", func(c *fiber.Ctx) error {
		//Convert the id to a UUID
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Warn(err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid UUID")
		}

		//Delete the data from the storage backend
		err = storageBackend.Delete(id)
		if err != nil {
			log.Error(err)
			return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
		}

		return c.JSON(fiber.Map{
			"status": "OK",
		})
	})

	app.Post("/encrypt", func(c *fiber.Ctx) error {
		c.Accepts("application/json")

		// Parse the JSON body from the request
		type RequestBody struct {
			Body            string `json:"body"`
			TTL             int64  `json:"ttl"`
			ClientEncrypted bool   `json:"client_encrypted"`
		}

		var body RequestBody
		err := c.BodyParser(&body)
		if err != nil {
			log.Error(err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid JSON")
		}

		// Encrypt and save the data
		id, err := encryptAndSave(body.Body, body.TTL, body.ClientEncrypted, encryptionBackend, storageBackend)

		if err != nil {
			log.Error(err)
			if errors.Is(err, errInvalidTTL) ||
				errors.Is(err, errSecretTooLong) ||
				errors.Is(err, errInvalidClientEncryptionEnvelope) {
				return c.Status(fiber.StatusBadRequest).SendString(err.Error())
			}
			return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
		}

		// Return a JSON response with the UUID
		return c.JSON(fiber.Map{
			"id": id,
		})
	})

	return app
}

func encryptAndSave(body string, ttl int64, clientEncrypted bool, encryption encryption.EncryptionBackend, storage storage.StorageBackend) (string, error) {
	// Check the TTL is in range
	currentTimestamp := time.Now().Unix()
	if ttl < currentTimestamp || ttl > currentTimestamp+(MAX_AGE_IN_DAYS*24*60*60) {
		log.Error("Invalid TTL")
		return "", errInvalidTTL
	}

	// Validate the body
	if len(body) > MAX_SECRET_LENGTH {
		log.Error("Secret too long")
		return "", errSecretTooLong
	}
	if clientEncrypted && !isClientEncryptionEnvelope(body) {
		log.Error("Invalid client encryption envelope")
		return "", errInvalidClientEncryptionEnvelope
	}

	// Encrypt the data
	encryptedData, key, err := encryption.Encrypt([]byte(body))
	if err != nil {
		return "", err
	}

	// Store the encrypted data
	id, err := storage.Store(encryptedData, key, ttl, clientEncrypted)
	if err != nil {
		return "", err
	}

	return id.String(), nil
}

type clientEncryptionEnvelope struct {
	Version    int    `json:"v"`
	KDF        string `json:"k"`
	Iterations int    `json:"i"`
	Salt       string `json:"s"`
	Cipher     string `json:"c"`
	Nonce      string `json:"n"`
	Data       string `json:"d"`
}

// isClientEncryptionEnvelope strictly recognizes the current browser format.
// New secrets carry an explicit storage marker; this parser is also the
// temporary fallback for Web Crypto links created before that marker existed.
func isClientEncryptionEnvelope(body string) bool {
	if !strings.HasPrefix(body, CLIENT_ENCRYPTION_PREFIX) {
		return false
	}

	decoder := json.NewDecoder(strings.NewReader(strings.TrimPrefix(body, CLIENT_ENCRYPTION_PREFIX)))
	decoder.DisallowUnknownFields()

	var envelope clientEncryptionEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return false
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return false
	}

	if envelope.Version != 2 ||
		envelope.KDF != "PBKDF2-SHA-256" ||
		envelope.Iterations != CLIENT_ENCRYPTION_ITERATIONS ||
		envelope.Cipher != "AES-256-GCM" {
		return false
	}

	salt, err := base64.RawURLEncoding.DecodeString(envelope.Salt)
	if err != nil || len(salt) != 16 {
		return false
	}
	nonce, err := base64.RawURLEncoding.DecodeString(envelope.Nonce)
	if err != nil || len(nonce) != 12 {
		return false
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(envelope.Data)
	return err == nil && len(ciphertext) >= 16
}

func getOtherLanguage(language string) string {
	if language == "en" {
		return "fr"
	} else {
		return "en"
	}
}

func loadLocales() map[string]map[string]string {

	locales := make([]string, 2)
	locales[0] = "en"
	locales[1] = "fr"

	translations := make(map[string]map[string]string)

	for _, locale := range locales {
		translations[locale] = make(map[string]string)
		file := fmt.Sprintf("./locales/%s.json", locale)

		byteValue, err := os.ReadFile(file)
		if err != nil {
			log.Fatal(err)
		}

		var result map[string]interface{}
		json.Unmarshal([]byte(byteValue), &result)

		for key, value := range result {
			translations[locale][key] = value.(string)
		}
	}

	return translations
}
