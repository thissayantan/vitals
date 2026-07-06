package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/thissayantan/vitals/internal/config"
	"github.com/thissayantan/vitals/internal/theme"
)

// runDoctor diagnoses the environment: color support, charset, optional tools,
// and config validity.
func runDoctor(_ []string) int {
	ok := true
	check := func(label, value string, good bool) {
		mark := "✓"
		if !good {
			mark = "✗"
			ok = false
		}
		fmt.Printf("  %s %-22s %s\n", mark, label, value)
	}
	note := func(label, value string) {
		fmt.Printf("  • %-22s %s\n", label, value)
	}

	fmt.Println("vitals doctor")
	fmt.Println("\nTerminal:")
	note("TERM", emptyDash(os.Getenv("TERM")))
	note("COLORTERM", emptyDash(os.Getenv("COLORTERM")))
	if _, noColor := os.LookupEnv("NO_COLOR"); noColor {
		note("NO_COLOR", "set (colors disabled)")
	}

	// Resolve the theme the way the renderer would, to report effective modes.
	cfg, cfgErr := config.Load()
	if cfg == nil {
		cfg = config.Defaults()
	}
	th, thErr := theme.Load(cfg.Theme, cfg.Charset, cfg.ColorMode, cfg.ThemeOverrides)
	if thErr == nil {
		check("color mode", colorModeName(th.Mode), th.Mode != theme.ModeNone)
		note("charset", charsetName(th.Charset))
		note("theme", cfg.Theme)
	} else {
		check("theme load", thErr.Error(), false)
	}

	fmt.Println("\nDependencies:")
	_, gitErr := exec.LookPath("git")
	check("git", boolText(gitErr == nil, "found", "not found (git segment hides)"), true)

	fmt.Println("\nConfig:")
	path := config.Discover()
	if path == "" {
		note("source", "built-in defaults (no config file)")
	} else {
		note("source", path)
		if data, err := os.ReadFile(path); err == nil {
			if verr := config.Validate(data); verr != nil {
				check("schema", "invalid", false)
				fmt.Printf("      %v\n", verr)
			} else {
				check("schema", "valid", true)
			}
		} else {
			check("readable", err.Error(), false)
		}
	}
	if cfgErr != nil {
		check("load", cfgErr.Error(), false)
	}

	fmt.Println()
	if ok {
		fmt.Println("All good.")
		return 0
	}
	fmt.Println("Some checks failed (see ✗ above).")
	return 1
}

func emptyDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func boolText(b bool, yes, no string) string {
	if b {
		return yes
	}
	return no
}

func colorModeName(m theme.ColorMode) string {
	switch m {
	case theme.ModeTrueColor:
		return "truecolor (24-bit)"
	case theme.ModeANSI256:
		return "ansi256"
	case theme.ModeANSI:
		return "ansi (16-color)"
	default:
		return "none"
	}
}

func charsetName(c theme.Charset) string {
	switch c {
	case theme.CharsetNerdFont:
		return "nerdfont"
	case theme.CharsetASCII:
		return "ascii"
	default:
		return "unicode"
	}
}
