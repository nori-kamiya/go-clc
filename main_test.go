package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout redirects os.Stdout for the duration of fn and returns what was
// written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()
	w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestRun_Dispatch(t *testing.T) {
	if code := run(nil); code != 2 {
		t.Errorf("no args: code = %d, want 2", code)
	}
	if code := run([]string{"bogus"}); code != 2 {
		t.Errorf("unknown cmd: code = %d, want 2", code)
	}

	var code int
	out := captureStdout(t, func() { code = run([]string{"version"}) })
	if code != 0 || !strings.Contains(out, "go-clc") {
		t.Errorf("version: code=%d out=%q", code, out)
	}

	out = captureStdout(t, func() { code = run([]string{"help"}) })
	if code != 0 || !strings.Contains(out, "使い方") {
		t.Errorf("help: code=%d out=%q", code, out)
	}
}

func TestIsTTY_Pipe(t *testing.T) {
	r, w, _ := os.Pipe()
	defer r.Close()
	defer w.Close()
	if isTTY(w) {
		t.Error("a pipe should not be reported as a TTY")
	}
}
