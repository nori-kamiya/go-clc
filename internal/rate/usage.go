package rate

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// UsageURL is the Claude Code OAuth usage endpoint.
const UsageURL = "https://api.anthropic.com/api/oauth/usage"

// FetchUsage performs the authenticated GET and returns the raw response body.
// HTTP and transport errors are translated into user-facing ClcError messages.
func FetchUsage(client *http.Client, url, token string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, &ClcError{Msg: fmt.Sprintf("リクエストの作成に失敗しました: %v", err)}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("User-Agent", "go-clc/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, &ClcError{Msg: fmt.Sprintf("使用量 API への接続に失敗しました: %v", err)}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, &ClcError{Msg: "認証に失敗しました (HTTP 401)。トークンが無効か失効している可能性があります。\n" +
			"  Claude Code を起動してトークンを更新してください。"}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := string(body)
		if len(msg) > 300 {
			msg = msg[:300]
		}
		return nil, &ClcError{Msg: fmt.Sprintf("使用量 API がエラーを返しました (HTTP %d): %s", resp.StatusCode, msg)}
	}
	return body, nil
}

// Window is a single rate-limit window in the usage response.
type Window struct {
	Utilization *float64        `json:"utilization"`
	ResetsAt    json.RawMessage `json:"resets_at"`
}

// ParseWindows extracts the top-level keys whose values are objects containing
// a "utilization" field, matching the Python comprehension.
func ParseWindows(body []byte) (map[string]Window, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(body, &top); err != nil {
		return nil, &ClcError{Msg: fmt.Sprintf("レスポンスの解析に失敗しました: %v", err)}
	}
	windows := make(map[string]Window, len(top))
	for k, raw := range top {
		var w Window
		if err := json.Unmarshal(raw, &w); err != nil {
			continue // not an object / shape mismatch -> skip, like the Python filter
		}
		if w.Utilization == nil {
			continue
		}
		windows[k] = w
	}
	return windows, nil
}
