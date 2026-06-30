package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/nori-kamiya/go-clc/internal/rate"
)

// version is overridable at build time: -ldflags "-X main.version=1.0.0".
// When unset (e.g. installed via `go install …@v0.0.1`), it falls back to the
// module version recorded in the build info.
var version = "dev"

func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}

const usage = `go-clc - Claude Code レート残量ビューア

使い方:
  go-clc rate [--json] [--no-color]   レート残量を表示する
  go-clc version                      バージョンを表示する
  go-clc help                         このヘルプを表示する

go-clc rate のオプション:
  --json        使用量 API の生レスポンスを JSON で表示
  --no-color    色付けを無効にする (NO_COLOR 環境変数でも可)
`

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}

	switch args[0] {
	case "rate":
		return cmdRate(args[1:])
	case "version", "--version", "-v":
		fmt.Printf("go-clc %s\n", resolveVersion())
		return 0
	case "help", "--help", "-h":
		fmt.Print(usage)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "エラー: 不明なコマンド %q\n\n%s", args[0], usage)
		return 2
	}
}

func cmdRate(args []string) int {
	fs := flag.NewFlagSet("rate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", false, "API の生レスポンスを JSON で表示")
	noColor := fs.Bool("no-color", false, "色付けを無効にする")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	opts, err := rate.DefaultOptions()
	if err != nil {
		fmt.Fprintf(os.Stderr, "エラー: %s\n", err)
		return 1
	}
	opts.JSON = *jsonOut
	opts.Stdout = os.Stdout
	opts.UseColor = !*jsonOut && !*noColor && os.Getenv("NO_COLOR") == "" && isTTY(os.Stdout)

	code, err := rate.Run(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "エラー: %s\n", err)
	}
	return code
}

// isTTY reports whether f is connected to a terminal (zero-dependency check).
func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
