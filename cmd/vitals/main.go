// Command vitals renders a modular status line for Claude Code.
//
// Modes (see DESIGN.md §3):
//
//	vitals                 read session JSON on stdin, render the status line (hot path)
//	vitals config          launch the TUI configurator (bubbletea) with live preview
//	vitals init            merge statusLine into ~/.claude/settings.json (backup first)
//	vitals doctor          diagnose color/font/dependency/config issues
//	vitals print [flags]   render from a file (dev helper: --input, --config)
//	vitals --version       print version
package main

import (
	"fmt"
	"io"
	"os"
)

// version is injected at build time via -ldflags "-X main.version=…".
var version = "dev"

func main() {
	args := os.Args[1:]

	// Subcommand dispatch. No subcommand ⇒ render mode.
	if len(args) > 0 {
		switch args[0] {
		case "config":
			os.Exit(runConfig(args[1:]))
		case "init":
			os.Exit(runInit(args[1:]))
		case "doctor":
			os.Exit(runDoctor(args[1:]))
		case "print":
			os.Exit(runPrint(args[1:]))
		case "--version", "-v", "version":
			fmt.Println("vitals", version)
			return
		case "--help", "-h", "help":
			usage(os.Stdout)
			return
		default:
			fmt.Fprintf(os.Stderr, "vitals: unknown command %q\n\n", args[0])
			usage(os.Stderr)
			os.Exit(2)
		}
	}

	os.Exit(runRender(os.Stdin, os.Stdout))
}

func usage(w io.Writer) {
	_, _ = fmt.Fprint(w, `vitals — a modular status line for Claude Code

Usage:
  vitals                 render the status line (reads session JSON on stdin)
  vitals config          customize segments, theme, and order (TUI)
  vitals init            wire vitals into ~/.claude/settings.json
  vitals doctor          diagnose setup problems
  vitals print [flags]   render from a file (--input, --config)
  vitals --version       print version
`)
}
