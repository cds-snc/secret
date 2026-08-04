package app

import (
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cds-snc/secret/encryption"
	"github.com/cds-snc/secret/storage"
	"github.com/gofiber/fiber/v2"
)

type failOnceEncryption struct {
	failed atomic.Bool
}

func (e *failOnceEncryption) Init(map[string]string) error {
	return nil
}

func (e *failOnceEncryption) Encrypt(plaintext []byte) ([]byte, []byte, error) {
	return plaintext, nil, nil
}

func (e *failOnceEncryption) Decrypt(ciphertext, _ []byte) ([]byte, error) {
	if e.failed.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("transient decryption failure")
	}
	return ciphertext, nil
}

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

func TestCreateAppRendersVersionedClientEncryption(t *testing.T) {
	t.Parallel()

	app := CreateApp(&encryption.NullEncryption{}, &storage.NullBackend{})
	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("CreateApp() GET / returned an error: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading GET / response: %v", err)
	}

	page := string(body)
	for _, expected := range []string{
		`const ENVELOPE_PREFIX = "emc:v2:"`,
		`const PBKDF2_ITERATIONS = 600000`,
		`name: "AES-GCM"`,
		`client_encrypted: clientEncrypted`,
	} {
		if !strings.Contains(page, expected) {
			t.Errorf("CreateApp() GET / did not render %q", expected)
		}
	}
	for _, removed := range []string{"CryptoJS", "decryptLegacy", "looksLikeLegacyEnvelope"} {
		if strings.Contains(page, removed) {
			t.Errorf("CreateApp() GET / still rendered removed legacy code %q", removed)
		}
	}
}

func TestCreateAppRendersLocalizedDecryptionError(t *testing.T) {
	t.Parallel()

	app := CreateApp(&encryption.NullEncryption{}, &storage.NullBackend{})
	req := httptest.NewRequest("GET", "/fr/view/00000000-0000-0000-0000-000000000000", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("CreateApp() GET French view returned an error: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading French view response: %v", err)
	}

	page := string(body)
	if !strings.Contains(page, "Le mot de passe est incorrect ou le message chiffré est endommagé.") {
		t.Error("CreateApp() did not render the localized decryption error")
	}
	if !strings.Contains(page, "Ce message a été chiffré avec un mot de passe additionnel.") {
		t.Error("CreateApp() did not render the localized encrypted-message notice")
	}
}

func TestCreateAppGetHomeWithOptionalAdditionalPassword(t *testing.T) {
	t.Parallel()

	app := CreateAppWithConfig(&encryption.NullEncryption{}, &storage.NullBackend{}, AppConfig{})

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("CreateApp() GET / = %v, want %v", resp.StatusCode, fiber.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyString := string(body)

	if !strings.Contains(bodyString, `label="Optional password"`) {
		t.Errorf("CreateApp() GET / = %v, want %v", bodyString, `label="Optional password"`)
	}

	if !strings.Contains(bodyString, `const requireAdditionalPassword = false;`) {
		t.Errorf("CreateApp() GET / = %v, want %v", bodyString, `const requireAdditionalPassword = false;`)
	}

	if strings.Contains(bodyString, `error-message="Enter an additional password"`) {
		t.Errorf("CreateApp() GET / should not require an additional password by default")
	}
}

func TestCreateAppGetHomeWithRequiredAdditionalPassword(t *testing.T) {
	t.Parallel()

	app := CreateAppWithConfig(
		&encryption.NullEncryption{},
		&storage.NullBackend{},
		AppConfig{RequireAdditionalPassword: true},
	)

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("CreateApp() GET / = %v, want %v", resp.StatusCode, fiber.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyString := string(body)

	if !strings.Contains(bodyString, `label="Additional password"`) {
		t.Errorf("CreateApp() GET / = %v, want %v", bodyString, `label="Additional password"`)
	}

	if !strings.Contains(bodyString, `error-message="Enter an additional password"`) {
		t.Errorf("CreateApp() GET / = %v, want %v", bodyString, `error-message="Enter an additional password"`)
	}

	if !strings.Contains(bodyString, `const requireAdditionalPassword = true;`) {
		t.Errorf("CreateApp() GET / = %v, want %v", bodyString, `const requireAdditionalPassword = true;`)
	}
}

func TestCreateAppGetHomeWithRequiredAdditionalPasswordInFrench(t *testing.T) {
	t.Parallel()

	app := CreateAppWithConfig(
		&encryption.NullEncryption{},
		&storage.NullBackend{},
		AppConfig{RequireAdditionalPassword: true},
	)

	req := httptest.NewRequest("GET", "/fr", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("CreateApp() GET /fr = %v, want %v", resp.StatusCode, fiber.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyString := string(body)

	if !strings.Contains(bodyString, `label="Mot de passe additionnel"`) {
		t.Errorf("CreateApp() GET /fr = %v, want %v", bodyString, `label="Mot de passe additionnel"`)
	}

	if !strings.Contains(bodyString, `error-message="Entrez un mot de passe additionnel"`) {
		t.Errorf("CreateApp() GET /fr = %v, want %v", bodyString, `error-message="Entrez un mot de passe additionnel"`)
	}

	if !strings.Contains(bodyString, `const requireAdditionalPassword = true;`) {
		t.Errorf("CreateApp() GET /fr = %v, want %v", bodyString, `const requireAdditionalPassword = true;`)
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
	if !strings.Contains(string(body), `class="d-none" id="decrypt-div"`) {
		t.Error("CreateApp() view page did not hide password controls by default")
	}
}

func TestCreateAppPostDecryptWithInvalidUUID(t *testing.T) {
	t.Parallel()

	app := CreateApp(&encryption.NullEncryption{}, &storage.NullBackend{})

	req := httptest.NewRequest("POST", "/decrypt/invalid-uuid", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("CreateApp() POST /decrypt/invalid-uuid = %v, want %v", resp.StatusCode, fiber.StatusBadRequest)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-store, max-age=0" {
		t.Errorf("Cache-Control = %q, want %q", got, "no-store, max-age=0")
	}
}

func TestCreateAppPostDecryptWithValidUUID(t *testing.T) {
	t.Parallel()

	backend := &storage.InMemoryStorageBackend{}
	backend.Init(map[string]string{})

	id, _ := backend.Store([]byte("test"), []byte("test"), time.Now().Add(time.Hour).Unix(), false)

	app := CreateApp(&encryption.NullEncryption{}, backend)

	req := httptest.NewRequest("POST", "/decrypt/"+id.String(), nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("CreateApp() POST /decrypt/valid-uuid = %v, want %v", resp.StatusCode, fiber.StatusOK)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-store, max-age=0" {
		t.Errorf("Cache-Control = %q, want %q", got, "no-store, max-age=0")
	}

	body, _ := io.ReadAll(resp.Body)

	//Check if the body contains the right JSON response
	if !strings.Contains(string(body), `"body":"test"`) ||
		!strings.Contains(string(body), `"client_encrypted":false`) {
		t.Errorf("CreateApp() POST /decrypt/valid-uuid returned unexpected body: %s", body)
	}

	// Check if the data was deleted from the storage backend
	_, err := backend.Claim(id, time.Minute)
	if !errors.Is(err, storage.ErrSecretUnavailable) {
		t.Errorf("Claim() after decrypt = %v, want ErrSecretUnavailable", err)
	}
}

func TestCreateAppGetDoesNotRetrieveSecret(t *testing.T) {
	t.Parallel()

	backend := &storage.InMemoryStorageBackend{}
	if err := backend.Init(map[string]string{}); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}
	id, err := backend.Store([]byte("test"), nil, time.Now().Add(time.Hour).Unix(), false)
	if err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	app := CreateApp(&encryption.NullEncryption{}, backend)
	resp, err := app.Test(httptest.NewRequest("GET", "/decrypt/"+id.String(), nil))
	if err != nil {
		t.Fatalf("GET /decrypt returned an error: %v", err)
	}
	if resp.StatusCode == fiber.StatusOK {
		t.Fatal("GET /decrypt unexpectedly retrieved the secret")
	}

	claim, err := backend.Claim(id, time.Minute)
	if err != nil {
		t.Fatalf("secret was consumed by GET /decrypt: %v", err)
	}
	if err := backend.Release(id, claim.Token); err != nil {
		t.Fatalf("Release() failed: %v", err)
	}
}

func TestCreateAppConcurrentDecryptHasExactlyOneWinner(t *testing.T) {
	t.Parallel()

	backend := &storage.InMemoryStorageBackend{}
	if err := backend.Init(map[string]string{}); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}
	id, err := backend.Store([]byte("test"), nil, time.Now().Add(time.Hour).Unix(), false)
	if err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	app := CreateApp(&encryption.NullEncryption{}, backend)
	const contenders = 32
	start := make(chan struct{})
	statuses := make(chan int, contenders)
	var wg sync.WaitGroup

	for range contenders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			req := httptest.NewRequest("POST", "/decrypt/"+id.String(), nil)
			resp, requestErr := app.Test(req)
			if requestErr != nil {
				t.Errorf("POST /decrypt returned an error: %v", requestErr)
				return
			}
			statuses <- resp.StatusCode
		}()
	}

	close(start)
	wg.Wait()
	close(statuses)

	successes := 0
	for status := range statuses {
		if status == fiber.StatusOK {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successful concurrent decryptions = %d, want exactly 1", successes)
	}
}

func TestCreateAppReleasesClaimAfterDecryptionFailure(t *testing.T) {
	t.Parallel()

	backend := &storage.InMemoryStorageBackend{}
	if err := backend.Init(map[string]string{}); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}
	id, err := backend.Store([]byte("test"), nil, time.Now().Add(time.Hour).Unix(), false)
	if err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	app := CreateApp(&failOnceEncryption{}, backend)
	first, _ := app.Test(httptest.NewRequest("POST", "/decrypt/"+id.String(), nil))
	if first.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("first decrypt status = %d, want %d", first.StatusCode, fiber.StatusBadRequest)
	}

	second, _ := app.Test(httptest.NewRequest("POST", "/decrypt/"+id.String(), nil))
	if second.StatusCode != fiber.StatusOK {
		t.Fatalf("retry decrypt status = %d, want %d", second.StatusCode, fiber.StatusOK)
	}

	third, _ := app.Test(httptest.NewRequest("POST", "/decrypt/"+id.String(), nil))
	if third.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("decrypt after successful retry status = %d, want %d", third.StatusCode, fiber.StatusBadRequest)
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

	backend := &storage.InMemoryStorageBackend{}
	backend.Init(map[string]string{})

	id, _ := backend.Store([]byte("test"), []byte("test"), time.Now().Add(time.Hour).Unix(), false)

	app := CreateApp(&encryption.NullEncryption{}, backend)

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
	_, err := backend.Claim(id, time.Minute)
	if !errors.Is(err, storage.ErrSecretUnavailable) {
		t.Errorf("Claim() after delete = %v, want ErrSecretUnavailable", err)
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

func TestCreateAppDecryptUsesStoredClientEncryptionMarker(t *testing.T) {
	t.Parallel()

	const envelope = `emc:v2:{"v":2,"k":"PBKDF2-SHA-256","i":600000,"s":"AAAAAAAAAAAAAAAAAAAAAA","c":"AES-256-GCM","n":"AAAAAAAAAAAAAAAA","d":"AAAAAAAAAAAAAAAAAAAAAA"}`

	tests := []struct {
		name            string
		body            string
		clientEncrypted bool
	}{
		{
			name:            "encrypted envelope",
			body:            envelope,
			clientEncrypted: true,
		},
		{
			name:            "plaintext that resembles an envelope",
			body:            envelope,
			clientEncrypted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &storage.InMemoryStorageBackend{}
			if err := backend.Init(map[string]string{}); err != nil {
				t.Fatalf("Init() failed: %v", err)
			}
			id, err := backend.Store([]byte(tt.body), nil, time.Now().Add(time.Hour).Unix(), tt.clientEncrypted)
			if err != nil {
				t.Fatalf("Store() failed: %v", err)
			}

			app := CreateApp(&encryption.NullEncryption{}, backend)
			resp, err := app.Test(httptest.NewRequest("POST", "/decrypt/"+id.String(), nil))
			if err != nil {
				t.Fatalf("POST /decrypt failed: %v", err)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("reading response: %v", err)
			}
			want := fmt.Sprintf(`"client_encrypted":%t`, tt.clientEncrypted)
			if !strings.Contains(string(body), want) {
				t.Fatalf("POST /decrypt response = %s, want %s", body, want)
			}
		})
	}
}

func TestClientEncryptionEnvelopeDetection(t *testing.T) {
	t.Parallel()

	const valid = `emc:v2:{"v":2,"k":"PBKDF2-SHA-256","i":600000,"s":"AAAAAAAAAAAAAAAAAAAAAA","c":"AES-256-GCM","n":"AAAAAAAAAAAAAAAA","d":"AAAAAAAAAAAAAAAAAAAAAA"}`
	tests := []struct {
		name string
		body string
		want bool
	}{
		{name: "valid current envelope", body: valid, want: true},
		{name: "prefix only", body: CLIENT_ENCRYPTION_PREFIX, want: false},
		{name: "wrong iterations", body: strings.Replace(valid, "600000", "128", 1), want: false},
		{name: "unknown field", body: strings.Replace(valid, `"v":2`, `"v":2,"extra":true`, 1), want: false},
		{name: "legacy CryptoJS payload", body: strings.Repeat("a", 64) + "Zm9v", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isClientEncryptionEnvelope(tt.body); got != tt.want {
				t.Fatalf("isClientEncryptionEnvelope() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestCreateAppRejectsInvalidMarkedClientEnvelope(t *testing.T) {
	t.Parallel()

	app := CreateApp(&encryption.NullEncryption{}, &storage.NullBackend{})
	ttl := fmt.Sprint(time.Now().Add(time.Hour).Unix())
	req := httptest.NewRequest(
		"POST",
		"/encrypt",
		strings.NewReader(`{"body":"not-an-envelope","client_encrypted":true,"ttl":`+ttl+`}`),
	)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("POST /encrypt status = %d, want %d", resp.StatusCode, fiber.StatusBadRequest)
	}
}

func TestCreateAppPostEncryptWithRequiredAdditionalPasswordFlag(t *testing.T) {
	t.Parallel()

	app := CreateAppWithConfig(
		&encryption.NullEncryption{},
		&storage.NullBackend{},
		AppConfig{RequireAdditionalPassword: true},
	)

	ttl := fmt.Sprint(time.Now().Add(time.Hour).Unix())

	req := httptest.NewRequest("POST", "/encrypt", strings.NewReader(`{"body":"test", "ttl":`+ttl+`}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("CreateApp() POST /encrypt = %v, want %v", resp.StatusCode, fiber.StatusOK)
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
