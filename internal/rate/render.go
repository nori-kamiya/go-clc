package rate

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

// pyRound replicates Python 3's round(): round-half-to-even (banker's rounding).
// Inputs here are always non-negative.
func pyRound(x float64) int {
	f := math.Floor(x)
	diff := x - f
	switch {
	case diff < 0.5:
		return int(f)
	case diff > 0.5:
		return int(f) + 1
	default: // exactly .5 -> round to even
		if int64(f)%2 == 0 {
			return int(f)
		}
		return int(f) + 1
	}
}

// PadDisplay right-pads text with spaces until its terminal display width
// reaches width (full/wide characters count as 2). Never truncates.
func PadDisplay(text string, width int) string {
	pad := width - displayWidth(text)
	if pad <= 0 {
		return text
	}
	return text + strings.Repeat(" ", pad)
}

// Colorize wraps text in an ANSI color escape chosen by utilization thresholds.
// Returns text unchanged when useColor is false.
func Colorize(text string, utilization float64, useColor bool, cfg Config) string {
	if !useColor {
		return text
	}
	var code string
	switch {
	case utilization >= cfg.ColorDangerThreshold:
		code = "31" // red
	case utilization >= cfg.ColorWarnThreshold:
		code = "33" // yellow
	default:
		code = "32" // green
	}
	return "\033[" + code + "m" + text + "\033[0m"
}

// ParseResetsAt converts a resets_at value (JSON number = epoch seconds, or
// ISO-8601 string) into a time.Time. ok is false when the value is null,
// absent, or unparseable. A naive ISO string is assumed to be UTC.
func ParseResetsAt(raw json.RawMessage) (time.Time, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return time.Time{}, false
	}
	// Numeric -> epoch seconds (may be fractional).
	var num float64
	if err := json.Unmarshal(raw, &num); err == nil {
		sec := int64(num)
		nsec := int64((num - float64(sec)) * 1e9)
		return time.Unix(sec, nsec).UTC(), true
	}
	// String -> ISO-8601.
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return time.Time{}, false
	}
	return parseISO(s)
}

func parseISO(s string) (time.Time, bool) {
	// Try layouts with explicit offset/zone first, then naive (assume UTC).
	withZone := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05Z07:00",
	}
	for _, l := range withZone {
		if t, err := time.Parse(l, s); err == nil {
			return t, true
		}
	}
	naive := []string{
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, l := range naive {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), true // naive == UTC, mirroring Python
		}
	}
	return time.Time{}, false
}

// FormatRemaining renders a remaining duration as "Nd,HHhMMm" (days/hours
// omitted when zero), clamped at zero. Matches the Python format_remaining.
func FormatRemaining(d time.Duration) string {
	totalMin := int(math.Floor(d.Seconds() / 60))
	if totalMin < 0 {
		totalMin = 0
	}
	days := totalMin / 1440
	rem := totalMin % 1440
	hours := rem / 60
	minutes := rem % 60

	var b strings.Builder
	if days > 0 {
		fmt.Fprintf(&b, "%dd,", days)
	}
	if hours > 0 {
		fmt.Fprintf(&b, "%02dh", hours)
	}
	fmt.Fprintf(&b, "%02dm", minutes)
	return b.String()
}

// RenderWindow renders one rate-limit window into its multi-line block.
// ok is false when the window has no utilization (skipped). now is used to
// compute the "resets in" countdown.
func RenderWindow(key string, w Window, useColor bool, cfg Config, now time.Time) (string, bool) {
	if w.Utilization == nil {
		return "", false
	}
	utilization := *w.Utilization

	label := cfg.WindowLabels[key]
	if label == "" {
		label = key
	}

	capped := utilization
	if capped > 100 {
		capped = 100
	}
	filled := pyRound(float64(cfg.BarWidth) * capped / 100)
	if filled < 0 {
		filled = 0
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", cfg.BarWidth-filled)
	bar = Colorize(bar, utilization, useColor, cfg)

	lines := []string{fmt.Sprintf("%s[%s] %5.1f%% used", PadDisplay(label, cfg.LabelWidth), bar, utilization)}

	indent := strings.Repeat(" ", cfg.LabelWidth)
	if resets, ok := ParseResetsAt(w.ResetsAt); ok {
		lines = append(lines, fmt.Sprintf("%sresets in %s", indent, FormatRemaining(resets.Sub(now))))
	} else if key == "extra_usage" {
		used := utilization / 100.0 * cfg.MonthlyLimitUSD
		lines = append(lines, fmt.Sprintf("%sused $%.2f", indent, used))
	}
	return strings.Join(lines, "\n"), true
}
