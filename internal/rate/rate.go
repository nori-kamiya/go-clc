package rate

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"time"
)

// Options bundles the injectable dependencies for the rate command, so the core
// logic can be unit-tested without touching the Keychain, network, or clock.
type Options struct {
	Keychain   KeychainFunc
	CredPath   string
	HTTPClient *http.Client
	UsageURL   string
	Cfg        Config
	Now        func() time.Time
	UseColor   bool
	JSON       bool
	Stdout     io.Writer
}

// DefaultOptions returns Options wired to the real environment.
func DefaultOptions() (Options, error) {
	cfg, err := LoadConfig(ConfigPath())
	if err != nil {
		return Options{}, err
	}
	return Options{
		Keychain:   SecurityKeychain,
		CredPath:   DefaultCredPath(),
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
		UsageURL:   UsageURL,
		Cfg:        cfg,
		Now:        time.Now,
	}, nil
}

// Run executes the rate command: resolve token, fetch usage, render. It returns
// a process exit code and an error (a *ClcError carries a user-facing message).
func Run(opts Options) (int, error) {
	now := opts.Now()
	token, err := GetAccessToken(opts.Keychain, opts.CredPath, now)
	if err != nil {
		return 1, err
	}

	body, err := FetchUsage(opts.HTTPClient, opts.UsageURL, token)
	if err != nil {
		return 1, err
	}

	if opts.JSON {
		var buf bytes.Buffer
		if err := json.Indent(&buf, body, "", "  "); err != nil {
			// Not valid JSON; emit raw so the user can still inspect it.
			opts.Stdout.Write(body)
			io.WriteString(opts.Stdout, "\n")
			return 0, nil
		}
		buf.WriteByte('\n')
		opts.Stdout.Write(buf.Bytes())
		return 0, nil
	}

	windows, err := ParseWindows(body)
	if err != nil {
		return 1, err
	}

	ordered := orderWindows(windows, opts.Cfg.WindowOrder)
	if len(ordered) == 0 {
		io.WriteString(opts.Stdout, "表示できるレート枠が見つかりませんでした。生のレスポンスは --json で確認できます。\n")
		return 1, nil
	}

	for i, key := range ordered {
		block, ok := RenderWindow(key, windows[key], opts.UseColor, opts.Cfg, now)
		if !ok {
			continue
		}
		io.WriteString(opts.Stdout, block)
		if i < len(ordered)-1 {
			io.WriteString(opts.Stdout, "\n\n")
		} else {
			io.WriteString(opts.Stdout, "\n")
		}
	}
	return 0, nil
}

// orderWindows returns keys in configured order first, then remaining keys
// sorted alphabetically — matching the Python ordering exactly.
func orderWindows(windows map[string]Window, order []string) []string {
	inOrder := make(map[string]bool, len(order))
	ordered := make([]string, 0, len(windows))
	for _, k := range order {
		if _, ok := windows[k]; ok {
			ordered = append(ordered, k)
			inOrder[k] = true
		}
	}
	rest := make([]string, 0, len(windows))
	for k := range windows {
		if !inOrder[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	return append(ordered, rest...)
}
