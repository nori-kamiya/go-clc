package rate

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestApplyOverrides_AllFields(t *testing.T) {
	p := t.TempDir() + "/c.json"
	writeFile(t, p, `{
		"bar_width":20,"label_width":10,"monthly_limit_usd":3.5,
		"color_warn_threshold":40,"color_danger_threshold":70,
		"window_labels":{"x":"X"},"window_order":["x","y"]
	}`)
	cfg, err := LoadConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BarWidth != 20 || cfg.LabelWidth != 10 || cfg.MonthlyLimitUSD != 3.5 ||
		cfg.ColorWarnThreshold != 40 || cfg.ColorDangerThreshold != 70 ||
		cfg.WindowLabels["x"] != "X" || strings.Join(cfg.WindowOrder, ",") != "x,y" {
		t.Errorf("override not fully applied: %+v", cfg)
	}
}

func TestConfigPath_HomeFallback(t *testing.T) {
	t.Setenv("CLC_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	got := ConfigPath()
	if !strings.HasSuffix(got, "/.config/go-clc/config.json") {
		t.Errorf("home fallback path = %s", got)
	}
}

func TestDefaultCredPath(t *testing.T) {
	if p := DefaultCredPath(); !strings.HasSuffix(p, "/.claude/.credentials.json") {
		t.Errorf("DefaultCredPath = %s", p)
	}
}

func TestDefaultOptions(t *testing.T) {
	t.Setenv("CLC_CONFIG", "") // avoid picking up a real override file
	opts, err := DefaultOptions()
	if err != nil {
		t.Fatal(err)
	}
	if opts.HTTPClient == nil || opts.HTTPClient.Timeout != 15*time.Second {
		t.Errorf("http client not configured: %+v", opts.HTTPClient)
	}
	if opts.UsageURL != UsageURL || opts.Keychain == nil || opts.Now == nil {
		t.Errorf("options not wired: %+v", opts)
	}
}

func TestSecurityKeychain_NotFound(t *testing.T) {
	// A service that does not exist -> found=false, no error (covers the darwin
	// branch and the security non-zero-exit handling).
	raw, found, err := SecurityKeychain("clc-test-nonexistent-service-xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found && raw == "" {
		t.Error("found=true but empty raw is inconsistent")
	}
}

func TestLoadCredentialBlob_FileReadError(t *testing.T) {
	// credPath is a directory -> ReadFile returns a non-NotExist error.
	_, err := loadCredentialBlob(nil, t.TempDir())
	if err == nil {
		t.Fatal("expected read error for directory path")
	}
}

func TestParseResetsAt_Fractional(t *testing.T) {
	ts, ok := ParseResetsAt(json.RawMessage(`1700000000.5`))
	if !ok || ts.Unix() != 1700000000 || ts.Nanosecond() == 0 {
		t.Errorf("fractional epoch failed: %v ok=%v ns=%d", ts, ok, ts.Nanosecond())
	}
}

func TestParseResetsAt_ObjectNotTime(t *testing.T) {
	if _, ok := ParseResetsAt(json.RawMessage(`{"a":1}`)); ok {
		t.Error("object should not parse as time")
	}
}

func TestRenderWindow_NegativeUtilClampedToEmpty(t *testing.T) {
	cfg := DefaultConfig()
	w := Window{Utilization: f(-5)}
	block, ok := RenderWindow("five_hour", w, false, cfg, time.Now())
	if !ok {
		t.Fatal("expected ok")
	}
	if !strings.Contains(block, strings.Repeat("░", cfg.BarWidth)) {
		t.Errorf("negative util should be all-empty bar: %q", block)
	}
}

func TestRun_JSONInvalidFallback(t *testing.T) {
	opts, cleanup := baseOpts(t, `this is not json`)
	defer cleanup()
	var out bytes.Buffer
	opts.Stdout = &out
	opts.JSON = true
	code, err := Run(opts)
	if code != 0 || err != nil {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if !strings.Contains(out.String(), "this is not json") {
		t.Errorf("raw fallback missing: %q", out.String())
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadConfig_ReadErrorDirectory(t *testing.T) {
	// A directory path -> ReadFile returns a non-NotExist error.
	if _, err := LoadConfig(t.TempDir()); err == nil {
		t.Error("expected read error for directory config path")
	}
}

func TestDefaultOptions_BadConfig(t *testing.T) {
	p := t.TempDir() + "/bad.json"
	writeFile(t, p, "{bad json")
	t.Setenv("CLC_CONFIG", p)
	if _, err := DefaultOptions(); err == nil {
		t.Error("expected error from bad config")
	}
}

func TestRun_FetchError(t *testing.T) {
	opts, cleanup := baseOpts(t, "")
	cleanup() // close server immediately -> connection refused
	var out bytes.Buffer
	opts.Stdout = &out
	code, err := Run(opts)
	if code != 1 || err == nil {
		t.Fatalf("expected fetch error, code=%d err=%v", code, err)
	}
}

func TestRun_ParseWindowsError(t *testing.T) {
	opts, cleanup := baseOpts(t, "not valid json")
	defer cleanup()
	var out bytes.Buffer
	opts.Stdout = &out
	code, err := Run(opts) // JSON mode off -> ParseWindows fails
	if code != 1 || err == nil {
		t.Fatalf("expected parse error, code=%d err=%v", code, err)
	}
}

func TestFetchUsage_BadURL(t *testing.T) {
	_, err := FetchUsage(newClient(), "http://invalid\x00url", "tok")
	if err == nil {
		t.Error("expected request creation error")
	}
}

func TestConfigPath_NoHome(t *testing.T) {
	t.Setenv("CLC_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	if got := ConfigPath(); got != "" {
		t.Errorf("expected empty path when home unresolved, got %q", got)
	}
}

func TestDefaultCredPath_NoHome(t *testing.T) {
	t.Setenv("HOME", "")
	if got := DefaultCredPath(); got != "" {
		t.Errorf("expected empty cred path when home unresolved, got %q", got)
	}
}

func TestClassifyKeychain(t *testing.T) {
	// success: trims trailing newline, found=true
	raw, found, err := classifyKeychain([]byte("secret-token\n"), nil)
	if err != nil || !found || raw != "secret-token" {
		t.Fatalf("success: raw=%q found=%v err=%v", raw, found, err)
	}
	// non-zero exit -> not found, no error
	exitErr := exec.Command("sh", "-c", "exit 1").Run()
	_, found, err = classifyKeychain(nil, exitErr)
	if found || err != nil {
		t.Fatalf("exit error: found=%v err=%v", found, err)
	}
	// other error -> propagated
	_, _, err = classifyKeychain(nil, errors.New("boom"))
	if err == nil {
		t.Fatal("generic error should propagate")
	}
}
