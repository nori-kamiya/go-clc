package rate

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPyRound_HalfToEven(t *testing.T) {
	cases := map[float64]int{
		0.5: 0, 1.5: 2, 2.5: 2, 6.5: 6, 3.5: 4,
		0.4: 0, 0.6: 1, 2.49: 2, 2.51: 3, 13.0: 13,
	}
	for in, want := range cases {
		if got := pyRound(in); got != want {
			t.Errorf("pyRound(%v) = %d, want %d", in, got, want)
		}
	}
}

func TestPadDisplay(t *testing.T) {
	cases := []struct {
		text  string
		width int
		want  string
	}{
		{"5Hours", 7, "5Hours "},
		{"Weekly", 7, "Weekly "},
		{"Weekly(Opus)", 7, "Weekly(Opus)"}, // longer than width: unchanged
		{"週次", 7, "週次   "},                  // 2 wide runes = width 4, pad 3 spaces
		{"", 3, "   "},
	}
	for _, c := range cases {
		if got := PadDisplay(c.text, c.width); got != c.want {
			t.Errorf("PadDisplay(%q,%d) = %q, want %q", c.text, c.width, got, c.want)
		}
	}
}

func TestDisplayWidth(t *testing.T) {
	if w := displayWidth("abc"); w != 3 {
		t.Errorf("ascii width = %d, want 3", w)
	}
	if w := displayWidth("週次"); w != 4 {
		t.Errorf("cjk width = %d, want 4", w)
	}
	if w := displayWidth("a週"); w != 3 {
		t.Errorf("mixed width = %d, want 3", w)
	}
}

func TestColorize(t *testing.T) {
	cfg := DefaultConfig()
	if got := Colorize("x", 90, false, cfg); got != "x" {
		t.Errorf("no-color should pass through, got %q", got)
	}
	if got := Colorize("x", 90, true, cfg); got != "\033[31mx\033[0m" {
		t.Errorf("danger = %q", got)
	}
	if got := Colorize("x", 50, true, cfg); got != "\033[33mx\033[0m" {
		t.Errorf("warn boundary = %q", got)
	}
	if got := Colorize("x", 49.9, true, cfg); got != "\033[32mx\033[0m" {
		t.Errorf("safe = %q", got)
	}
	if got := Colorize("x", 80, true, cfg); got != "\033[31mx\033[0m" {
		t.Errorf("danger boundary = %q", got)
	}
}

func TestFormatRemaining(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{45 * time.Minute, "45m"},
		{0, "00m"},
		{-5 * time.Minute, "00m"}, // clamped
		{2*time.Hour + 5*time.Minute, "02h05m"},
		{26 * time.Hour, "1d,02h00m"},
		{(2*1440 + 3*60 + 7) * time.Minute, "2d,03h07m"},
		{90 * time.Second, "01m"}, // floor to minutes
		{119 * time.Second, "01m"},
	}
	for _, c := range cases {
		if got := FormatRemaining(c.d); got != c.want {
			t.Errorf("FormatRemaining(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestParseResetsAt(t *testing.T) {
	// epoch seconds (number)
	if ts, ok := ParseResetsAt(json.RawMessage(`1700000000`)); !ok || ts.Unix() != 1700000000 {
		t.Errorf("numeric epoch failed: %v ok=%v", ts, ok)
	}
	// ISO with Z
	ts, ok := ParseResetsAt(json.RawMessage(`"2026-01-02T03:04:05Z"`))
	if !ok || ts.UTC().Format(time.RFC3339) != "2026-01-02T03:04:05Z" {
		t.Errorf("iso Z failed: %v ok=%v", ts, ok)
	}
	// ISO with offset
	ts, ok = ParseResetsAt(json.RawMessage(`"2026-01-02T12:00:00+09:00"`))
	if !ok || ts.UTC().Format(time.RFC3339) != "2026-01-02T03:00:00Z" {
		t.Errorf("iso offset failed: %v ok=%v", ts, ok)
	}
	// naive ISO assumed UTC
	ts, ok = ParseResetsAt(json.RawMessage(`"2026-01-02T03:04:05"`))
	if !ok || ts.UTC().Format(time.RFC3339) != "2026-01-02T03:04:05Z" {
		t.Errorf("iso naive failed: %v ok=%v", ts, ok)
	}
	// null / empty / garbage
	if _, ok := ParseResetsAt(json.RawMessage(`null`)); ok {
		t.Error("null should be not-ok")
	}
	if _, ok := ParseResetsAt(nil); ok {
		t.Error("nil should be not-ok")
	}
	if _, ok := ParseResetsAt(json.RawMessage(`"not-a-date"`)); ok {
		t.Error("garbage should be not-ok")
	}
}

func f(v float64) *float64 { return &v }

func TestRenderWindow_WithResets(t *testing.T) {
	cfg := DefaultConfig()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	resets, _ := json.Marshal(now.Add(2*time.Hour + 30*time.Minute).Unix())
	w := Window{Utilization: f(50.0), ResetsAt: resets}

	block, ok := RenderWindow("five_hour", w, false, cfg, now)
	if !ok {
		t.Fatal("expected ok")
	}
	// 13*50/100 = 6.5 -> banker's rounding -> 6 filled, 7 empty.
	wantBlock := "5Hours [██████░░░░░░░]  50.0% used\n       resets in 02h30m"
	if block != wantBlock {
		t.Errorf("block =\n%q\nwant\n%q", block, wantBlock)
	}
}

func TestRenderWindow_ExtraUsageNoResets(t *testing.T) {
	cfg := DefaultConfig()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	w := Window{Utilization: f(25.0)} // no resets_at
	block, ok := RenderWindow("extra_usage", w, false, cfg, now)
	if !ok {
		t.Fatal("expected ok")
	}
	want := "Extra  [███░░░░░░░░░░]  25.0% used\n       used $2.50"
	if block != want {
		t.Errorf("block =\n%q\nwant\n%q", block, want)
	}
}

func TestRenderWindow_Over100Capped(t *testing.T) {
	cfg := DefaultConfig()
	now := time.Now()
	w := Window{Utilization: f(150.0)}
	block, _ := RenderWindow("five_hour", w, false, cfg, now)
	want := "5Hours [█████████████] 150.0% used"
	if block != want {
		t.Errorf("block = %q, want %q", block, want)
	}
}

func TestRenderWindow_NilUtilSkipped(t *testing.T) {
	if _, ok := RenderWindow("x", Window{}, false, DefaultConfig(), time.Now()); ok {
		t.Error("nil utilization should be skipped")
	}
}
