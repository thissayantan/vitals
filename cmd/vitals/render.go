package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/sayantan/vitals/internal/claude"
	"github.com/sayantan/vitals/internal/config"
	"github.com/sayantan/vitals/internal/render"
	"github.com/sayantan/vitals/internal/theme"
)

// runRender is the hot path: parse session JSON from r, render, print to w.
// It must never break the user's status bar, so every failure degrades to a
// minimal fallback line and exit 0.
func runRender(r io.Reader, w io.Writer) int {
	sess, err := claude.Parse(r)
	if err != nil {
		_, _ = fmt.Fprintln(w, fallbackLine())
		return 0
	}
	out, err := renderSession(sess, "")
	if err != nil {
		_, _ = fmt.Fprintln(w, fallbackLine())
		return 0
	}
	_, _ = fmt.Fprintln(w, out)
	return 0
}

// renderSession loads config + theme and renders the session. configPath
// overrides discovery when non-empty.
func renderSession(sess *claude.Session, configPath string) (string, error) {
	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		cfg = config.Defaults()
	}
	th, err := theme.Load(cfg.Theme, cfg.Charset, cfg.ColorMode, cfg.ThemeOverrides)
	if err != nil {
		return "", err
	}
	return render.Render(sess, cfg, th), nil
}

// fallbackLine is shown when parsing/rendering fails entirely.
func fallbackLine() string {
	return "─ Claude Code ─"
}

// runPrint is a dev helper: render from a file (--input) with an optional
// --config override. Defaults --input to testdata/session.sample.json.
func runPrint(args []string) int {
	fs := flag.NewFlagSet("print", flag.ContinueOnError)
	input := fs.String("input", "testdata/session.sample.json", "session JSON file to render")
	cfgPath := fs.String("config", "", "config file to use (overrides discovery)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	f, err := os.Open(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vitals print: %v\n", err)
		return 1
	}
	defer func() { _ = f.Close() }()

	sess, err := claude.Parse(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vitals print: parse: %v\n", err)
		return 1
	}
	out, err := renderSession(sess, *cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vitals print: render: %v\n", err)
		return 1
	}
	fmt.Println(out)
	return 0
}
