package rate

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fakeKeychain(raw string, found bool, err error) KeychainFunc {
	return func(service string) (string, bool, error) {
		if service != KeychainService {
			return "", false, nil
		}
		return raw, found, err
	}
}

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), ".credentials.json")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

const validCreds = `{"claudeAiOauth":{"accessToken":"tok-abc","expiresAt":4102444800000}}`

func TestGetAccessToken_FromKeychain(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tok, err := GetAccessToken(fakeKeychain(validCreds, true, nil), "", now)
	if err != nil || tok != "tok-abc" {
		t.Fatalf("tok=%q err=%v", tok, err)
	}
}

func TestGetAccessToken_FallbackToFile(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	path := writeTemp(t, validCreds)
	// keychain reports not found -> should read file
	tok, err := GetAccessToken(fakeKeychain("", false, nil), path, now)
	if err != nil || tok != "tok-abc" {
		t.Fatalf("tok=%q err=%v", tok, err)
	}
}

func TestGetAccessToken_KeychainPreferredOverFile(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	path := writeTemp(t, `{"claudeAiOauth":{"accessToken":"from-file","expiresAt":4102444800000}}`)
	tok, err := GetAccessToken(fakeKeychain(validCreds, true, nil), path, now)
	if err != nil || tok != "tok-abc" {
		t.Fatalf("expected keychain token, got tok=%q err=%v", tok, err)
	}
}

func TestGetAccessToken_NotFound(t *testing.T) {
	now := time.Now()
	_, err := GetAccessToken(fakeKeychain("", false, nil), filepath.Join(t.TempDir(), "missing.json"), now)
	var ce *ClcError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ClcError, got %v", err)
	}
}

func TestGetAccessToken_Expired(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	// expiresAt = 2020-01-01 in ms
	expired := `{"claudeAiOauth":{"accessToken":"tok","expiresAt":1577836800000}}`
	_, err := GetAccessToken(fakeKeychain(expired, true, nil), "", now)
	var ce *ClcError
	if !errors.As(err, &ce) {
		t.Fatalf("expected expiry ClcError, got %v", err)
	}
}

func TestGetAccessToken_NoExpiryFieldOK(t *testing.T) {
	now := time.Now()
	tok, err := GetAccessToken(fakeKeychain(`{"claudeAiOauth":{"accessToken":"t"}}`, true, nil), "", now)
	if err != nil || tok != "t" {
		t.Fatalf("expiresAt absent should pass: tok=%q err=%v", tok, err)
	}
}

func TestGetAccessToken_MissingToken(t *testing.T) {
	now := time.Now()
	_, err := GetAccessToken(fakeKeychain(`{"claudeAiOauth":{"expiresAt":4102444800000}}`, true, nil), "", now)
	var ce *ClcError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ClcError for missing token, got %v", err)
	}
}

func TestGetAccessToken_MalformedJSON(t *testing.T) {
	now := time.Now()
	_, err := GetAccessToken(fakeKeychain(`{not json`, true, nil), "", now)
	var ce *ClcError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ClcError for malformed json, got %v", err)
	}
}

func TestGetAccessToken_KeychainError(t *testing.T) {
	now := time.Now()
	_, err := GetAccessToken(fakeKeychain("", false, errors.New("boom")), "", now)
	if err == nil {
		t.Fatal("expected error to propagate")
	}
}

func TestGetAccessToken_EmptyKeychainFallsThrough(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	path := writeTemp(t, validCreds)
	// keychain returns found=true but blank -> should fall back to file
	tok, err := GetAccessToken(fakeKeychain("   ", true, nil), path, now)
	if err != nil || tok != "tok-abc" {
		t.Fatalf("tok=%q err=%v", tok, err)
	}
}
