package main

import (
	"fmt"
	"os"

	"github.com/thissayantan/vitals/internal/tui"
)

// runConfig launches the TUI configurator (internal/tui) with a live preview.
func runConfig(_ []string) int {
	if err := tui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "vitals config: %v\n", err)
		return 1
	}
	return 0
}
