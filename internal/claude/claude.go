// Package claude defines the types for the JSON object Claude Code pipes to the
// status line on stdin, plus defensive parsing. No business logic lives here.
//
// See DESIGN.md §2 for the field contract. Parse defensively: every field is
// optional and a missing value must let the corresponding segment hide itself.
package claude

import (
	"encoding/json"
	"io"
	"path/filepath"
	"regexp"
	"strings"
)

// Session is the JSON object Claude Code pipes to the status line on stdin.
// Every field is optional; absent values stay at their zero value so segments
// can hide themselves. Fields that must distinguish "absent" from "zero" use
// pointers (see Limit.UsedPercentage).
type Session struct {
	Model          Model         `json:"model"`
	ContextWindow  ContextWindow `json:"context_window"`
	Cost           Cost          `json:"cost"`
	Workspace      Workspace     `json:"workspace"`
	RateLimits     RateLimits    `json:"rate_limits"`
	Worktree       Worktree      `json:"worktree"`
	TranscriptPath string        `json:"transcript_path"`
	SessionID      string        `json:"session_id"`
	CWD            string        `json:"cwd"`
}

// Model identifies the active model.
type Model struct {
	DisplayName string `json:"display_name"`
	ID          string `json:"id"`
}

// ContextWindow describes context-window usage.
type ContextWindow struct {
	UsedPercentage    float64 `json:"used_percentage"`
	ContextWindowSize int64   `json:"context_window_size"`
}

// Cost holds session economics and edit stats.
type Cost struct {
	TotalCostUSD      float64 `json:"total_cost_usd"`
	TotalDurationMs   int64   `json:"total_duration_ms"`
	TotalLinesAdded   int64   `json:"total_lines_added"`
	TotalLinesRemoved int64   `json:"total_lines_removed"`
}

// Workspace describes the directories of the active session.
type Workspace struct {
	CurrentDir string `json:"current_dir"`
	ProjectDir string `json:"project_dir"`
}

// RateLimits holds the subscription rate-limit windows.
type RateLimits struct {
	FiveHour Limit `json:"five_hour"`
	SevenDay Limit `json:"seven_day"`
}

// Limit is one rate-limit window. UsedPercentage is a pointer so an absent
// window (nil) is distinguishable from a real 0% — the prototype treats absence
// as "hide", matching jq's `// -1` default.
type Limit struct {
	UsedPercentage *float64 `json:"used_percentage"`
	ResetsAt       int64    `json:"resets_at"`
}

// Worktree describes the active git worktree, if any.
type Worktree struct {
	Branch string `json:"branch"`
	Name   string `json:"name"`
}

// contextSuffix matches a trailing " (… context)" annotation on a display name,
// e.g. "Opus 4.8 (1M context)" → "Opus 4.8".
var contextSuffix = regexp.MustCompile(` *\(.*context\)$`)

// Parse reads and decodes the session JSON from r. It returns an error only for
// malformed JSON; missing fields are left at their zero value so callers (and
// segments) degrade gracefully. A nil/empty stream yields a zero Session.
func Parse(r io.Reader) (*Session, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var s Session
	if len(strings.TrimSpace(string(data))) == 0 {
		return &s, nil
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// DisplayModel returns the model name with any "(… context)" suffix stripped,
// falling back to "Claude" when no name is present.
func (s *Session) DisplayModel() string {
	name := contextSuffix.ReplaceAllString(s.Model.DisplayName, "")
	name = strings.TrimSpace(name)
	if name == "" {
		return "Claude"
	}
	return name
}

// CurrentDir resolves the working directory: workspace.current_dir, then cwd,
// then ".".
func (s *Session) CurrentDir() string {
	if s.Workspace.CurrentDir != "" {
		return s.Workspace.CurrentDir
	}
	if s.CWD != "" {
		return s.CWD
	}
	return "."
}

// ProjectDir resolves the project root: workspace.project_dir, then CurrentDir.
func (s *Session) ProjectDir() string {
	if s.Workspace.ProjectDir != "" {
		return s.Workspace.ProjectDir
	}
	return s.CurrentDir()
}

// DirName is the basename of the current directory, never empty.
func (s *Session) DirName() string {
	base := filepath.Base(s.CurrentDir())
	if base == "" || base == "." {
		return "/"
	}
	return base
}

// ResolvedSessionID returns session_id, falling back to the transcript file's
// basename (without .jsonl) — matching the prototype's task-dir resolution.
func (s *Session) ResolvedSessionID() string {
	if s.SessionID != "" {
		return s.SessionID
	}
	if s.TranscriptPath != "" {
		base := filepath.Base(s.TranscriptPath)
		return strings.TrimSuffix(base, ".jsonl")
	}
	return ""
}
