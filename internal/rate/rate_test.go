package rate

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func baseOpts(t *testing.T, responseBody string) (Options, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(responseBody))
	}))
	opts := Options{
		Keychain:   fakeKeychain(validCreds, true, nil),
		CredPath:   "",
		HTTPClient: newClient(),
		UsageURL:   srv.URL,
		Cfg:        DefaultConfig(),
		Now:        func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
		UseColor:   false,
		Stdout:     &bytes.Buffer{},
	}
	return opts, srv.Close
}

func TestRun_RendersOrderedWindows(t *testing.T) {
	body := `{
		"seven_day":{"utilization":40},
		"five_hour":{"utilization":50},
		"extra_usage":{"utilization":25}
	}`
	opts, cleanup := baseOpts(t, body)
	defer cleanup()
	var out bytes.Buffer
	opts.Stdout = &out

	code, err := Run(opts)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	got := out.String()
	// five_hour must come before seven_day (config order), extra_usage last.
	iFive := strings.Index(got, "5Hours")
	iWeek := strings.Index(got, "Weekly")
	iExtra := strings.Index(got, "Extra")
	if !(iFive < iWeek && iWeek < iExtra) {
		t.Errorf("ordering wrong:\n%s", got)
	}
	if !strings.Contains(got, "used $2.50") {
		t.Errorf("extra usage line missing:\n%s", got)
	}
	// blocks separated by blank line, trailing single newline.
	if !strings.HasSuffix(got, "used $2.50\n") {
		t.Errorf("trailing format wrong: %q", got)
	}
}

func TestRun_UnknownKeysSortedAfter(t *testing.T) {
	body := `{
		"zeta_window":{"utilization":1},
		"alpha_window":{"utilization":1},
		"five_hour":{"utilization":1}
	}`
	opts, cleanup := baseOpts(t, body)
	defer cleanup()
	var out bytes.Buffer
	opts.Stdout = &out
	Run(opts)
	got := out.String()
	iFive := strings.Index(got, "5Hours")
	iAlpha := strings.Index(got, "alpha_window")
	iZeta := strings.Index(got, "zeta_window")
	if !(iFive < iAlpha && iAlpha < iZeta) {
		t.Errorf("expected configured-first then sorted rest:\n%s", got)
	}
}

func TestRun_JSONMode(t *testing.T) {
	body := `{"five_hour":{"utilization":10}}`
	opts, cleanup := baseOpts(t, body)
	defer cleanup()
	var out bytes.Buffer
	opts.Stdout = &out
	opts.JSON = true

	code, err := Run(opts)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	got := out.String()
	if !strings.Contains(got, "\"five_hour\"") || !strings.Contains(got, "  ") {
		t.Errorf("expected indented json, got: %s", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Error("json should end with newline")
	}
}

func TestRun_NoWindows(t *testing.T) {
	opts, cleanup := baseOpts(t, `{"meta":"none"}`)
	defer cleanup()
	var out bytes.Buffer
	opts.Stdout = &out
	code, err := Run(opts)
	if code != 1 || err != nil {
		t.Fatalf("expected code 1, got %d err=%v", code, err)
	}
	if !strings.Contains(out.String(), "レート枠が見つかりません") {
		t.Errorf("expected no-data message, got %q", out.String())
	}
}

func TestRun_TokenError(t *testing.T) {
	opts, cleanup := baseOpts(t, `{}`)
	defer cleanup()
	opts.Keychain = fakeKeychain("", false, nil)
	opts.CredPath = "" // nothing found
	code, err := Run(opts)
	if code != 1 || err == nil {
		t.Fatalf("expected token error, code=%d err=%v", code, err)
	}
}

func TestOrderWindows(t *testing.T) {
	ws := map[string]Window{"five_hour": {}, "b": {}, "a": {}, "seven_day": {}}
	got := orderWindows(ws, DefaultConfig().WindowOrder)
	want := []string{"five_hour", "seven_day", "a", "b"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("orderWindows = %v, want %v", got, want)
	}
}
