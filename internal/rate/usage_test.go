package rate

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newClient() *http.Client { return &http.Client{Timeout: 5 * time.Second} }

func TestFetchUsage_OKAndHeaders(t *testing.T) {
	var gotAuth, gotBeta, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotBeta = r.Header.Get("anthropic-beta")
		gotUA = r.Header.Get("User-Agent")
		w.Write([]byte(`{"five_hour":{"utilization":10}}`))
	}))
	defer srv.Close()

	body, err := FetchUsage(newClient(), srv.URL, "tok-123")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "five_hour") {
		t.Errorf("body = %s", body)
	}
	if gotAuth != "Bearer tok-123" {
		t.Errorf("auth header = %q", gotAuth)
	}
	if gotBeta != "oauth-2025-04-20" {
		t.Errorf("beta header = %q", gotBeta)
	}
	if gotUA != "go-clc/1.0" {
		t.Errorf("ua header = %q", gotUA)
	}
}

func TestFetchUsage_401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	_, err := FetchUsage(newClient(), srv.URL, "tok")
	var ce *ClcError
	if !errors.As(err, &ce) || !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected 401 ClcError, got %v", err)
	}
}

func TestFetchUsage_500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	}))
	defer srv.Close()
	_, err := FetchUsage(newClient(), srv.URL, "tok")
	if err == nil || !strings.Contains(err.Error(), "HTTP 500") {
		t.Fatalf("expected 500 error, got %v", err)
	}
}

func TestFetchUsage_NetworkError(t *testing.T) {
	// Closed server -> connection refused.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()
	_, err := FetchUsage(newClient(), url, "tok")
	if err == nil || !strings.Contains(err.Error(), "接続に失敗") {
		t.Fatalf("expected network error, got %v", err)
	}
}

func TestParseWindows_FiltersNonObjectsAndNoUtil(t *testing.T) {
	body := []byte(`{
		"five_hour":{"utilization":12.5,"resets_at":1700000000},
		"seven_day":{"utilization":40},
		"object_no_util":{"foo":1},
		"scalar":42,
		"text":"hello",
		"nullval":null
	}`)
	ws, err := ParseWindows(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(ws) != 2 {
		t.Fatalf("expected 2 windows, got %d: %v", len(ws), keys(ws))
	}
	if ws["five_hour"].Utilization == nil || *ws["five_hour"].Utilization != 12.5 {
		t.Errorf("five_hour util wrong: %+v", ws["five_hour"])
	}
}

func TestParseWindows_BadJSON(t *testing.T) {
	if _, err := ParseWindows([]byte(`not json`)); err == nil {
		t.Error("expected error")
	}
}

func keys(m map[string]Window) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
