package rate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// KeychainService is the macOS Keychain generic-password service name Claude
// Code stores its OAuth credentials under.
const KeychainService = "Claude Code-credentials"

// KeychainFunc reads a credential blob from a secure store by service name.
// It returns (raw, found, err). found is false when no entry exists (not an
// error). Injectable so tests need not touch the real Keychain.
type KeychainFunc func(service string) (raw string, found bool, err error)

// SecurityKeychain reads from the macOS login keychain via the `security` CLI.
// On non-darwin platforms it always reports "not found".
func SecurityKeychain(service string) (string, bool, error) {
	if runtime.GOOS != "darwin" {
		return "", false, nil
	}
	out, err := exec.Command("security", "find-generic-password", "-s", service, "-w").Output()
	return classifyKeychain(out, err)
}

// classifyKeychain interprets the output/error of the `security` CLI.
// A non-zero exit means the item is absent (not an error); any other failure
// (e.g. the binary missing) propagates.
func classifyKeychain(out []byte, err error) (string, bool, error) {
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return "", false, nil
		}
		return "", false, err
	}
	return strings.TrimRight(string(out), "\n"), true, nil
}

// DefaultCredPath returns ~/.claude/.credentials.json.
func DefaultCredPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", ".credentials.json")
}

type oauthCreds struct {
	ClaudeAiOauth struct {
		AccessToken string `json:"accessToken"`
		ExpiresAt   int64  `json:"expiresAt"` // epoch milliseconds
	} `json:"claudeAiOauth"`
}

// loadCredentialBlob returns the raw credentials JSON, preferring the Keychain
// and falling back to the credentials file.
func loadCredentialBlob(kc KeychainFunc, credPath string) (string, error) {
	if kc != nil {
		raw, found, err := kc(KeychainService)
		if err != nil {
			return "", err
		}
		if found && strings.TrimSpace(raw) != "" {
			return raw, nil
		}
	}
	if credPath != "" {
		data, err := os.ReadFile(credPath)
		if err == nil {
			return string(data), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", &ClcError{Msg: fmt.Sprintf(
		"認証トークンを取得できませんでした。\n"+
			"  確認した場所: macOS Keychain (%s) / %s\n"+
			"  Claude Code にサインインしているか確認してください。",
		KeychainService, credPath)}
}

// GetAccessToken resolves a valid OAuth access token, checking expiry against now.
func GetAccessToken(kc KeychainFunc, credPath string, now time.Time) (string, error) {
	raw, err := loadCredentialBlob(kc, credPath)
	if err != nil {
		return "", err
	}
	var creds oauthCreds
	if err := json.Unmarshal([]byte(raw), &creds); err != nil {
		return "", &ClcError{Msg: fmt.Sprintf("認証データの読み取りに失敗しました: %v", err)}
	}
	token := creds.ClaudeAiOauth.AccessToken
	if token == "" {
		return "", &ClcError{Msg: "accessToken フィールドが空です。Claude Code にサインインし直してください。"}
	}
	if creds.ClaudeAiOauth.ExpiresAt > 0 {
		expiry := time.UnixMilli(creds.ClaudeAiOauth.ExpiresAt)
		if expiry.Before(now) {
			return "", &ClcError{Msg: "アクセストークンが失効しています。\n" +
				"  Claude Code を起動すると自動的に更新されます。"}
		}
	}
	return token, nil
}
