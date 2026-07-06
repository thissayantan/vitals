package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// runInit merges the statusLine block into ~/.claude/settings.json, backing up
// the existing file first and preserving all other keys. Idempotent; never
// blind-overwrites.
//
// Flags:
//
//	--settings <path>   operate on a specific settings.json (default ~/.claude/settings.json)
//	--bin <path>        the vitals binary path to wire (default: this executable)
//	--dry-run           print the resulting JSON without writing
func runInit(args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	settingsPath := fs.String("settings", "", "settings.json path (default ~/.claude/settings.json)")
	binPath := fs.String("bin", "", "vitals binary path to wire (default: this executable)")
	dryRun := fs.Bool("dry-run", false, "print the result without writing")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	path := *settingsPath
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "vitals init: cannot find home dir: %v\n", err)
			return 1
		}
		path = filepath.Join(home, ".claude", "settings.json")
	}

	bin := *binPath
	if bin == "" {
		if exe, err := os.Executable(); err == nil {
			bin = exe
		} else {
			bin = "vitals"
		}
	}
	if abs, err := filepath.Abs(bin); err == nil {
		bin = abs
	}

	result, backup, err := mergeStatusLine(path, bin, *dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vitals init: %v\n", err)
		return 1
	}

	if *dryRun {
		fmt.Println(result)
		return 0
	}

	fmt.Printf("✓ wired vitals into %s\n", path)
	if backup != "" {
		fmt.Printf("  backup: %s\n", backup)
	}
	fmt.Println("\nNext steps:")
	fmt.Println("  • restart Claude Code (or start a new session) to see the status line")
	fmt.Println("  • run `vitals config` to customize segments, theme, and order")
	return 0
}

// mergeStatusLine reads settings.json (if present), deep-sets the statusLine
// block to invoke bin, writes a timestamped backup, and writes the merged
// result atomically. Returns the pretty-printed result and the backup path.
func mergeStatusLine(path, bin string, dryRun bool) (result, backup string, err error) {
	settings := map[string]any{}
	existing, readErr := os.ReadFile(path)
	hadFile := readErr == nil
	if hadFile && len(existing) > 0 {
		if err := json.Unmarshal(existing, &settings); err != nil {
			return "", "", fmt.Errorf("parse %s (fix or move it aside): %w", path, err)
		}
	}

	settings["statusLine"] = map[string]any{
		"type":    "command",
		"command": bin,
		"padding": 0,
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", "", err
	}
	out = append(out, '\n')
	if dryRun {
		return string(out), "", nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", "", err
	}

	// Back up an existing file before overwriting.
	if hadFile {
		backup = fmt.Sprintf("%s.bak.%d", path, time.Now().Unix())
		if err := os.WriteFile(backup, existing, 0o644); err != nil {
			return "", "", fmt.Errorf("write backup: %w", err)
		}
	}

	// Atomic write: temp file + rename.
	tmp, err := os.CreateTemp(filepath.Dir(path), "settings-*.json")
	if err != nil {
		return "", "", err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(out); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return "", "", err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return "", "", err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return "", "", err
	}
	return string(out), backup, nil
}
