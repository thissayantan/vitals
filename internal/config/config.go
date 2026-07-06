// Package config loads, merges, and validates vitals configuration.
//
// See DESIGN.md §5. Discovery precedence (first found wins, deep-merged onto
// built-in Defaults()):
//
//  1. ./.vitals.json                       (project-local)
//  2. $XDG_CONFIG_HOME/vitals/config.json  (user)
//  3. built-in Defaults()                  (the redesigned two-line layout)
//
// $VITALS_CONFIG overrides discovery with an explicit path.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thissayantan/vitals/internal/theme"
)

// SchemaURL is the published JSON-Schema URL written into saved configs.
const SchemaURL = "https://raw.githubusercontent.com/thissayantan/vitals/main/schema/vitals.schema.json"

// Config is the full vitals configuration.
type Config struct {
	Schema         string                 `json:"$schema,omitempty"`
	Theme          string                 `json:"theme"`
	Charset        string                 `json:"charset"`
	ColorMode      string                 `json:"colorMode"`
	Separator      string                 `json:"separator"`
	Lines          []Line                 `json:"lines"`
	ThemeOverrides map[string]theme.Style `json:"themeOverrides,omitempty"`
}

// Line is one status line: an ordered list of segments.
type Line struct {
	Segments []SegmentConfig `json:"segments"`
	Align    string          `json:"align,omitempty"`
}

// SegmentConfig is one segment entry. Enabled is a pointer so an omitted value
// defaults to true (IsEnabled).
type SegmentConfig struct {
	Type    string         `json:"type"`
	Enabled *bool          `json:"enabled,omitempty"`
	Style   *theme.Style   `json:"style,omitempty"`
	Options map[string]any `json:"options,omitempty"`
}

// IsEnabled reports whether the segment should render (default true).
func (sc SegmentConfig) IsEnabled() bool {
	return sc.Enabled == nil || *sc.Enabled
}

// Defaults returns the built-in redesigned two-line layout (DESIGN.md §4).
func Defaults() *Config {
	return &Config{
		Schema:    SchemaURL,
		Theme:     "catppuccin-mocha",
		Charset:   "auto",
		ColorMode: "auto",
		Separator: " │ ",
		Lines: []Line{
			{Segments: []SegmentConfig{
				{Type: "model"},
				{Type: "context", Options: map[string]any{"display": "both", "barWidth": 10}},
				{Type: "block"},
				{Type: "weekly"},
				{Type: "duration"},
				{Type: "cost", Options: map[string]any{"source": "auto"}},
			}},
			{Segments: []SegmentConfig{
				{Type: "directory", Options: map[string]any{"style": "basename"}},
				{Type: "worktree"},
				{Type: "git", Options: map[string]any{"showSha": false}},
				{Type: "diff"},
				{Type: "runtime"},
				{Type: "tasks"},
			}},
		},
	}
}

// LoadFrom loads config from an explicit path, deep-merged onto Defaults(). An
// empty path falls back to discovery (Load), so callers can pass a --config
// override through without a global side-channel. A read error yields Defaults()
// plus the error (the caller may ignore it and use Defaults()).
func LoadFrom(path string) (*Config, error) {
	if path == "" {
		return Load()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Defaults(), fmt.Errorf("read %s: %w", path, err)
	}
	return Merge(Defaults(), data)
}

// Load discovers, parses, and deep-merges the first config file found onto
// Defaults(). A missing/empty config yields Defaults(). Parse errors are
// returned (the caller may fall back to Defaults()).
func Load() (*Config, error) {
	path := Discover()
	if path == "" {
		return Defaults(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		// An explicitly-set but unreadable config is worth surfacing; otherwise
		// fall back silently to defaults.
		if os.Getenv("VITALS_CONFIG") != "" {
			return Defaults(), fmt.Errorf("read %s: %w", path, err)
		}
		return Defaults(), nil
	}
	return Merge(Defaults(), data)
}

// Discover returns the path of the config to load, or "" if none exists.
func Discover() string {
	if p := os.Getenv("VITALS_CONFIG"); p != "" {
		return p
	}
	if p := ".vitals.json"; fileExists(p) {
		return p
	}
	if p := UserConfigPath(); fileExists(p) {
		return p
	}
	return ""
}

// UserConfigPath is $XDG_CONFIG_HOME/vitals/config.json (default
// ~/.config/vitals/config.json), or "" if the home dir can't be resolved. It is
// both the user-level discovery location and the TUI's save target, so the two
// can never drift.
func UserConfigPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "vitals", "config.json")
}

// Merge deep-merges the JSON in data onto base and returns the result. Objects
// merge recursively; arrays and scalars replace. This means a user-supplied
// `lines` replaces the default lines wholesale (segment lists are not merged
// element-wise), while `themeOverrides` keys merge.
func Merge(base *Config, data []byte) (*Config, error) {
	var baseMap map[string]any
	bb, _ := json.Marshal(base)
	if err := json.Unmarshal(bb, &baseMap); err != nil {
		return nil, err
	}
	var userMap map[string]any
	if err := json.Unmarshal(data, &userMap); err != nil {
		return nil, fmt.Errorf("parse config JSON: %w", err)
	}
	merged := deepMerge(baseMap, userMap)

	mb, err := json.Marshal(merged)
	if err != nil {
		return nil, err
	}
	var out Config
	if err := json.Unmarshal(mb, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// deepMerge merges src onto dst recursively (objects merge, everything else
// replaces) and returns dst.
func deepMerge(dst, src map[string]any) map[string]any {
	for k, sv := range src {
		if dv, ok := dst[k]; ok {
			dm, dok := dv.(map[string]any)
			sm, sok := sv.(map[string]any)
			if dok && sok {
				dst[k] = deepMerge(dm, sm)
				continue
			}
		}
		dst[k] = sv
	}
	return dst
}

// Save writes the config to path as pretty JSON with the $schema set, creating
// parent directories as needed.
func (c *Config) Save(path string) error {
	if c.Schema == "" {
		c.Schema = SchemaURL
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func fileExists(p string) bool {
	if p == "" {
		return false
	}
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}
