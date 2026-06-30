package rate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_MissingFileReturnsDefaults(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BarWidth != 13 || cfg.LabelWidth != 7 || cfg.MonthlyLimitUSD != 10.0 {
		t.Errorf("defaults not applied: %+v", cfg)
	}
}

func TestLoadConfig_EmptyPath(t *testing.T) {
	cfg, err := LoadConfig("")
	if err != nil || cfg.BarWidth != 13 {
		t.Fatalf("empty path should give defaults: %+v err=%v", cfg, err)
	}
}

func TestLoadConfig_ShallowOverride(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(p, []byte(`{"bar_width":20,"monthly_limit_usd":5}`), 0o600)
	cfg, err := LoadConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BarWidth != 20 {
		t.Errorf("bar_width override failed: %d", cfg.BarWidth)
	}
	if cfg.MonthlyLimitUSD != 5 {
		t.Errorf("monthly override failed: %v", cfg.MonthlyLimitUSD)
	}
	// untouched keys keep defaults
	if cfg.LabelWidth != 7 {
		t.Errorf("label_width should remain default: %d", cfg.LabelWidth)
	}
	if cfg.WindowLabels["five_hour"] != "5Hours" {
		t.Errorf("default labels should remain: %v", cfg.WindowLabels)
	}
}

func TestLoadConfig_WindowLabelsFullyReplaced(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(p, []byte(`{"window_labels":{"five_hour":"5H"}}`), 0o600)
	cfg, _ := LoadConfig(p)
	if cfg.WindowLabels["five_hour"] != "5H" {
		t.Errorf("override label missing: %v", cfg.WindowLabels)
	}
	// shallow replace: other labels are gone
	if _, ok := cfg.WindowLabels["seven_day"]; ok {
		t.Errorf("shallow merge should replace whole map, got %v", cfg.WindowLabels)
	}
}

func TestLoadConfig_BadJSON(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(p, []byte(`{bad`), 0o600)
	if _, err := LoadConfig(p); err == nil {
		t.Error("expected error for bad json")
	}
}

func TestConfigPath_EnvOverride(t *testing.T) {
	t.Setenv("CLC_CONFIG", "/custom/path.json")
	if ConfigPath() != "/custom/path.json" {
		t.Errorf("CLC_CONFIG not honored: %s", ConfigPath())
	}
}

func TestConfigPath_XDG(t *testing.T) {
	t.Setenv("CLC_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	want := filepath.Join("/xdg", "go-clc", "config.json")
	if ConfigPath() != want {
		t.Errorf("XDG path = %s, want %s", ConfigPath(), want)
	}
}
