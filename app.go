package app

import (
	"encoding/json"
	"fmt"
	"os"
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

func CreateApp(encryption encryption.EncryptionBackend, storage storage.StorageBackend) *fiber.App {
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
			"Lang":      "en",
			"OtherLang": getOtherLanguage("en"),
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

	app.Get("/:language", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"Lang":      c.Params("language"),
			"OtherLang": getOtherLanguage(c.Params("language")),
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
			"Lang":      c.Params("language"),
			"OtherLang": getOtherLanguage(c.Params("language")),
			"ViewId":    id,
		}, "base")
	})

	app.Get("/decrypt/:id", func(c *fiber.Ctx) error {
		//Convert the id to a UUID
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Warn(err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid UUID")
		}

		//Get the encrypted data from the storage backend
		encryptedData, key, err := storage.Retrieve(id)
		if err != nil {
			log.Warn(err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid UUID")
		}

		//Decrypt the data
		decryptedData, err := encryption.Decrypt(encryptedData, key)
		if err != nil {
			log.Error(err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid UUID")
		}

		//Delete the data from the storage backend
		err = storage.Delete(id)
		if err != nil {
			log.Error(err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid UUID")
		}

		// Return a JSON response with the decrypted data
		return c.JSON(fiber.Map{
			"body": string(decryptedData),
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
		err = storage.Delete(id)
		if err != nil {
			log.Error(err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid UUID")
		}

		return c.JSON(fiber.Map{
			"status": "OK",
		})
	})

	app.Post("/encrypt", func(c *fiber.Ctx) error {
		c.Accepts("application/json")

		// Parse the JSON body from the request
		type RequestBody struct {
			Body string `json:"body"`
			TTL  int64  `json:"ttl"`
		}

		var body RequestBody
		err := c.BodyParser(&body)
		if err != nil {
			log.Error(err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid JSON")
		}

		// Encrypt and save the data
		id, err := encryptAndSave(body.Body, body.TTL, encryption, storage)

		if err != nil {
			log.Error(err)
			return c.Status(fiber.StatusBadRequest).SendString(err.Error())
		}

		// Return a JSON response with the UUID
		return c.JSON(fiber.Map{
			"id": id,
		})
	})

	return app
}

func encryptAndSave(body string, ttl int64, encryption encryption.EncryptionBackend, storage storage.StorageBackend) (string, error) {
	// Check the TTL is in range
	currentTimestamp := time.Now().Unix()
	if ttl < currentTimestamp || ttl > currentTimestamp+(MAX_AGE_IN_DAYS*24*60*60) {
		log.Error("Invalid TTL")
		return "", fmt.Errorf("invalid TTL")
	}

	// Validate the body
	if len(body) > MAX_SECRET_LENGTH {
		log.Error("Secret too long")
		return "", fmt.Errorf("secret too long")
	}

	// Encrypt the data
	encryptedData, key, err := encryption.Encrypt([]byte(body))
	if err != nil {
		return "", err
	}

	// Store the encrypted data
	id, err := storage.Store(encryptedData, key, ttl)
	if err != nil {
		return "", err
	}

	return id.String(), nil
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
