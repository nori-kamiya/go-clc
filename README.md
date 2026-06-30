# go-clc — Claude Code rate limit viewer

`go-clc rate` shows your Claude Code rate-limit usage across windows (5-hour,
weekly, weekly per-model, extra paid usage) as colored bars.

A self-contained, single-binary Go tool — no runtime required, just drop the
binary on your `PATH`.

```
5Hours [░░░░░░░░░░░░░]   2.0% used
       resets in 04h56m

Weekly [███████░░░░░░]  55.0% used
       resets in 1d,20h36m
```

## Install

```sh
go install github.com/nori-kamiya/go-clc@latest
```

Or build locally:

```sh
git clone https://github.com/nori-kamiya/go-clc
cd go-clc
go build -ldflags "-X main.version=$(git describe --tags 2>/dev/null || echo dev)" -o go-clc .
```

## Usage

```sh
go-clc rate              # colored bars (color auto-disabled when output is not a TTY)
go-clc rate --json       # raw usage API response, pretty-printed
go-clc rate --no-color   # force plain output (also honors the NO_COLOR env var)
go-clc version
go-clc help
```

## How it works

1. Reads the Claude Code OAuth token from the macOS Keychain
   (`security find-generic-password -s "Claude Code-credentials"`),
   falling back to `~/.claude/.credentials.json`. Expired tokens are rejected.
2. Calls `GET https://api.anthropic.com/api/oauth/usage`
   with `Authorization: Bearer …` and `anthropic-beta: oauth-2025-04-20`.
3. Renders each window's `utilization` as a bar plus a `resets in …` countdown
   (or `used $…` for the extra-usage window).

The token never leaves your machine except in the `Authorization` header sent
to Anthropic's own API.

## Configuration

Sensible defaults are built in. To customize, create a JSON file at
`~/.config/go-clc/config.json` (or point `$CLC_CONFIG` / `$XDG_CONFIG_HOME` at one).
Any top-level key you provide replaces that default; omitted keys keep their
defaults. See [`config.example.json`](config.example.json).

| key | default | meaning |
|---|---|---|
| `bar_width` | `13` | bar length in characters |
| `label_width` | `7` | left-column label width (East-Asian-width aware) |
| `monthly_limit_usd` | `10.0` | basis for the extra-usage `used $…` line |
| `color_warn_threshold` | `50` | ≥ this % → yellow |
| `color_danger_threshold` | `80` | ≥ this % → red |
| `window_labels` | see example | display name per window key |
| `window_order` | see example | render order; unlisted windows follow, sorted |

## Development

```sh
go test ./...                  # full suite
go test -cover ./internal/rate # coverage
```

The core logic in `internal/rate` is written against injected dependencies
(Keychain reader, HTTP client, clock), so it is unit-tested without touching
the real Keychain, network, or system clock.

## Credits

Inspired by [masa0902dev/claude-code-rate](https://github.com/masa0902dev/claude-code-rate)
([Zenn article](https://zenn.dev/masa0902dev/articles/clc-claude-code-rate)),
a Python CLI for the same Claude Code usage endpoint. `go-clc` is an
independent reimplementation in Go with its own code, wording, and structure.

## License

[MIT](LICENSE) © 2026 Nori Kamiya
