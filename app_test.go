package app

import (
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cds-snc/secret/encryption"
	"github.com/cds-snc/secret/storage"
	"github.com/gofiber/fiber/v2"
)

func TestCreateApp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
	}{
		{
			name: "Create App",
			want: "App",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreateApp(&encryption.NullEncryption{}, &storage.NullBackend{})
			if got == nil {
				t.Errorf("CreateApp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateAppGetHome(t *testing.T) {
	t.Parallel()

	app := CreateApp(&encryption.NullEncryption{}, &storage.NullBackend{})

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("CreateApp() GET / = %v, want %v", resp.StatusCode, fiber.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)

	// Check if the body contains the correct string
	if !strings.Contains(string(body), "generate-div") {
		t.Errorf("CreateApp() GET / = %v, want %v", string(body), "generate-div")
	}
}

func TestCreateAppGetVersionWithGitShaSetAndWithout(t *testing.T) {
	t.Parallel()

	os.Setenv("GIT_SHA", "test")

	app := CreateApp(&encryption.NullEncryption{}, &storage.NullBackend{})

	req := httptest.NewRequest("GET", "/version", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("CreateApp() GET /version = %v, want %v", resp.StatusCode, fiber.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)

	want := fmt.Sprintf(`{"version":"%s"}`, "test")

	//Check if the body contains the right JSON response
	if string(body) != want {
		t.Errorf("CreateApp() GET /version = %v, want %v", string(body), want)
	}

	os.Unsetenv("GIT_SHA")

	app = CreateApp(&encryption.NullEncryption{}, &storage.NullBackend{})

	req = httptest.NewRequest("GET", "/version", nil)
	resp, _ = app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("CreateApp() GET /version = %v, want %v", resp.StatusCode, fiber.StatusOK)
	}

	body, _ = io.ReadAll(resp.Body)

	want = fmt.Sprintf(`{"version":"%s"}`, "dev")

	//Check if the body contains the right JSON response
	if string(body) != want {
		t.Errorf("CreateApp() GET /version = %v, want %v", string(body), want)
	}
}

func TestCreateAppGetHomeWithLanguage(t *testing.T) {
	t.Parallel()

	app := CreateApp(&encryption.NullEncryption{}, &storage.NullBackend{})

	req := httptest.NewRequest("GET", "/fr", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("CreateApp() GET /fr = %v, want %v", resp.StatusCode, fiber.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)

	// Check if the body contains the correct string
	if !strings.Contains(string(body), "generate-div") {
		t.Errorf("CreateApp() GET /fr = %v, want %v", string(body), "generate-div")
	}

	// Check if the body contains the correct string for a language switch
	if !strings.Contains(string(body), `lang-href="/en"`) {
		t.Errorf("CreateApp() GET /fr = %v, want %v", string(body), `lang-href="/en"`)
	}
}

func TestCreateAppGetViewWithIvalidUUID(t *testing.T) {
	t.Parallel()

	app := CreateApp(&encryption.NullEncryption{}, &storage.NullBackend{})

	req := httptest.NewRequest("GET", "/en/view/invalid-uuid", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("CreateApp() GET /en/view/invalid-uuid = %v, want %v", resp.StatusCode, fiber.StatusBadRequest)
	}
}

func TestCreateAppGetViewWithValidUUID(t *testing.T) {
	t.Parallel()

	app := CreateApp(&encryption.NullEncryption{}, &storage.NullBackend{})

	req := httptest.NewRequest("GET", "/en/view/00000000-0000-0000-0000-000000000000", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("CreateApp() GET /en/view/00000000-0000-0000-0000-000000000000 = %v, want %v", resp.StatusCode, fiber.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)

	// Check if the body contains the correct string
	if !strings.Contains(string(body), "confirm-div") {
		t.Errorf("CreateApp() GET /en/view/00000000-0000-0000-0000-000000000000 = %v, want %v", string(body), "confirm-div")
	}
}

func TestCreateAppGetDecryptWithIvalidUUID(t *testing.T) {
	t.Parallel()

	app := CreateApp(&encryption.NullEncryption{}, &storage.NullBackend{})

	req := httptest.NewRequest("GET", "/decrypt/invalid-uuid", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("CreateApp() GET /decrypt/invalid-uuid = %v, want %v", resp.StatusCode, fiber.StatusBadRequest)
	}
}

func TestCreateAppGetDecryptWithValidUUID(t *testing.T) {
	t.Parallel()

	storage := &storage.InMemoryStorageBackend{}
	storage.Init(map[string]string{})

	id, _ := storage.Store([]byte("test"), []byte("test"), time.Now().Add(time.Hour).Unix())

	app := CreateApp(&encryption.NullEncryption{}, storage)

	req := httptest.NewRequest("GET", "/decrypt/"+id.String(), nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("CreateApp() GET /decrypt/valid-uuid = %v, want %v", resp.StatusCode, fiber.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)

	//Check if the body contains the right JSON response
	if !strings.Contains(string(body), `{"body":"test"}`) {
		t.Errorf("CreateApp() GET /decrypt/valid-uuid = %v, want %v", string(body), `{"body":"test"}`)
	}

	// Check if the data was deleted from the storage backend
	_, _, err := storage.Retrieve(id)
	if err == nil {
		t.Errorf("CreateApp() GET /decrypt/valid-uuid = %v, want %v", err, "error")
	}
}

func TestCreateAppDeleteInvalidUUID(t *testing.T) {
	t.Parallel()

	app := CreateApp(&encryption.NullEncryption{}, &storage.NullBackend{})

	req := httptest.NewRequest("DELETE", "/delete/invalid-uuid", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("CreateApp() DELETE /delete/invalid-uuid = %v, want %v", resp.StatusCode, fiber.StatusBadRequest)
	}
}

func TestCreateAppDeleteValidUUID(t *testing.T) {
	t.Parallel()

	storage := &storage.InMemoryStorageBackend{}
	storage.Init(map[string]string{})

	id, _ := storage.Store([]byte("test"), []byte("test"), time.Now().Add(time.Hour).Unix())

	app := CreateApp(&encryption.NullEncryption{}, storage)

	req := httptest.NewRequest("DELETE", "/delete/"+id.String(), nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("CreateApp() DELETE /delete/valid-uuid = %v, want %v", resp.StatusCode, fiber.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)

	//Check if the body contains the right JSON response
	if !strings.Contains(string(body), `{"status":"OK"}`) {
		t.Errorf("CreateApp() DELETE /delete/valid-uuid = %v, want %v", string(body), `{"status":"OK"}`)
	}

	// Check if the data was deleted from the storage backend
	_, _, err := storage.Retrieve(id)
	if err == nil {
		t.Errorf("CreateApp() DELETE /delete/valid-uuid = %v, want %v", err, "error")
	}
}

func TestCreateAppPostEncrypt(t *testing.T) {
	t.Parallel()

	app := CreateApp(&encryption.NullEncryption{}, &storage.NullBackend{})

	ttl := fmt.Sprint(time.Now().Add(time.Hour).Unix())

	req := httptest.NewRequest("POST", "/encrypt", strings.NewReader(`{"body":"test", "ttl":`+ttl+`}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("CreateApp() POST /encrypt = %v, want %v", resp.StatusCode, fiber.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)

	//Check if the body contains a UUID id
	if !strings.Contains(string(body), `"id":"`) {
		t.Errorf("CreateApp() POST /encrypt = %v, want %v", string(body), `"id":"`)
	}
}

func TestCreateAppPostEncryptWithInvalidJSON(t *testing.T) {
	t.Parallel()

	app := CreateApp(&encryption.NullEncryption{}, &storage.NullBackend{})

	req := httptest.NewRequest("POST", "/encrypt", strings.NewReader(`{"body":"test", "ttl":`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("CreateApp() POST /encrypt = %v, want %v", resp.StatusCode, fiber.StatusBadRequest)
	}
}

func TestCreateAppPostEncryptWithInvalidBody(t *testing.T) {
	t.Parallel()

	app := CreateApp(&encryption.NullEncryption{}, &storage.NullBackend{})

	req := httptest.NewRequest("POST", "/encrypt", strings.NewReader(`{"body":"`+strings.Repeat("a", MAX_SECRET_LENGTH+1)+`", "ttl":1000}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("CreateApp() POST /encrypt = %v, want %v", resp.StatusCode, fiber.StatusBadRequest)
	}
}

func TestCreateAppPostEncryptWithInvalidTTL(t *testing.T) {
	t.Parallel()

	app := CreateApp(&encryption.NullEncryption{}, &storage.NullBackend{})

	req := httptest.NewRequest("POST", "/encrypt", strings.NewReader(`{"body":"test", "ttl":0}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("CreateApp() POST /encrypt = %v, want %v", resp.StatusCode, fiber.StatusBadRequest)
	}
}

func TestGetOtherLanguage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		language string
		want     string
	}{
		{
			name:     "English",
			language: "en",
			want:     "fr",
		},
		{
			name:     "French",
			language: "fr",
			want:     "en",
		},
		{
			name:     "Spanish",
			language: "es",
			want:     "en",
		},
		{
			name:     "German",
			language: "de",
			want:     "en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getOtherLanguage(tt.language)
			if got != tt.want {
				t.Errorf("getOtherLanguage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadLocales(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want int
	}{
		{
			name: "Load Locales",
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := loadLocales()
			if len(got) != tt.want {
				t.Errorf("loadLocales() = %v, want %v", len(got), tt.want)
			}
		})
	}
}
