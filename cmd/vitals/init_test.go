package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMergeStatusLinePreservesKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	// An existing settings.json with unrelated keys that must survive.
	existing := map[string]any{
		"model": "opus",
		"hooks": map[string]any{"PreToolUse": []any{"echo hi"}},
		"statusLine": map[string]any{
			"type": "command", "command": "old-statusline", "padding": 2,
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	_, backup, err := mergeStatusLine(path, "/usr/local/bin/vitals", false)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}

	// Backup created and equal to the original bytes.
	if backup == "" {
		t.Fatal("expected a backup path")
	}
	bdata, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(bdata) != string(data) {
		t.Error("backup should equal original contents")
	}

	// Result: statusLine rewired, other keys intact.
	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if got["model"] != "opus" {
		t.Error("model key should be preserved")
	}
	if _, ok := got["hooks"]; !ok {
		t.Error("hooks key should be preserved")
	}
	sl, ok := got["statusLine"].(map[string]any)
	if !ok {
		t.Fatal("statusLine missing")
	}
	if sl["command"] != "/usr/local/bin/vitals" {
		t.Errorf("command = %v, want vitals path", sl["command"])
	}
	if sl["type"] != "command" {
		t.Errorf("type = %v", sl["type"])
	}
	if int(sl["padding"].(float64)) != 0 {
		t.Errorf("padding = %v, want 0", sl["padding"])
	}
}

func TestMergeStatusLineFreshFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "settings.json") // dir doesn't exist yet

	_, backup, err := mergeStatusLine(path, "vitals", false)
	if err != nil {
		t.Fatalf("merge fresh: %v", err)
	}
	if backup != "" {
		t.Error("no backup expected for a fresh file")
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("settings.json should be created: %v", err)
	}
}
