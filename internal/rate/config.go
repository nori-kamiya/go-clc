package rate

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds all tunable rendering/behavior settings. Defaults mirror the
// original Python clc tool; users may override any top-level key via a JSON
// config file (see ConfigPath).
type Config struct {
	BarWidth             int               `json:"bar_width"`
	LabelWidth           int               `json:"label_width"`
	MonthlyLimitUSD      float64           `json:"monthly_limit_usd"`
	ColorWarnThreshold   float64           `json:"color_warn_threshold"`
	ColorDangerThreshold float64           `json:"color_danger_threshold"`
	WindowLabels         map[string]string `json:"window_labels"`
	WindowOrder          []string          `json:"window_order"`
}

// DefaultConfig returns the built-in defaults (identical to the Python _DEFAULTS).
func DefaultConfig() Config {
	return Config{
		BarWidth:             13,
		LabelWidth:           7,
		MonthlyLimitUSD:      10.0,
		ColorWarnThreshold:   50,
		ColorDangerThreshold: 80,
		WindowLabels: map[string]string{
			"five_hour":            "5Hours",
			"seven_day":            "Weekly",
			"seven_day_opus":       "Weekly(Opus)",
			"seven_day_sonnet":     "Weekly(Sonnet)",
			"seven_day_oauth_apps": "Weekly(OAuth Apps)",
			"extra_usage":          "Extra",
		},
		WindowOrder: []string{
			"five_hour",
			"seven_day",
			"seven_day_opus",
			"seven_day_sonnet",
			"seven_day_oauth_apps",
			"extra_usage",
		},
	}
}

// configFile uses pointer fields so we can detect which top-level keys are
// present in the override file. This reproduces Python's shallow dict.update:
// a provided key replaces the whole default value (no deep merge).
type configFile struct {
	BarWidth             *int               `json:"bar_width"`
	LabelWidth           *int               `json:"label_width"`
	MonthlyLimitUSD      *float64           `json:"monthly_limit_usd"`
	ColorWarnThreshold   *float64           `json:"color_warn_threshold"`
	ColorDangerThreshold *float64           `json:"color_danger_threshold"`
	WindowLabels         *map[string]string `json:"window_labels"`
	WindowOrder          *[]string          `json:"window_order"`
}

// ConfigPath returns the location of the override config file. It honors
// $CLC_CONFIG, otherwise defaults to $XDG_CONFIG_HOME/go-clc/config.json
// (falling back to ~/.config/go-clc/config.json).
func ConfigPath() string {
	if p := os.Getenv("CLC_CONFIG"); p != "" {
		return p
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "go-clc", "config.json")
}

// LoadConfig returns DefaultConfig merged (shallowly) with overrides read from
// path. A missing file is not an error. An empty path skips loading entirely.
func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	var ov configFile
	if err := json.Unmarshal(data, &ov); err != nil {
		return cfg, err
	}
	applyOverrides(&cfg, ov)
	return cfg, nil
}

func applyOverrides(cfg *Config, ov configFile) {
	if ov.BarWidth != nil {
		cfg.BarWidth = *ov.BarWidth
	}
	if ov.LabelWidth != nil {
		cfg.LabelWidth = *ov.LabelWidth
	}
	if ov.MonthlyLimitUSD != nil {
		cfg.MonthlyLimitUSD = *ov.MonthlyLimitUSD
	}
	if ov.ColorWarnThreshold != nil {
		cfg.ColorWarnThreshold = *ov.ColorWarnThreshold
	}
	if ov.ColorDangerThreshold != nil {
		cfg.ColorDangerThreshold = *ov.ColorDangerThreshold
	}
	if ov.WindowLabels != nil {
		cfg.WindowLabels = *ov.WindowLabels
	}
	if ov.WindowOrder != nil {
		cfg.WindowOrder = *ov.WindowOrder
	}
}
