package config

// Presets are named starting layouts a user can seed (`vitals init --preset`)
// or cycle in the configurator. They set only the layout + look; discovery and
// merge behave the same as any config.

// PresetNames returns the preset names in a stable, cycle-friendly order.
func PresetNames() []string { return []string{"full", "minimal", "compact"} }

// Preset returns a named preset (a fresh *Config), or nil if unknown.
func Preset(name string) *Config {
	switch name {
	case "full":
		return Defaults()
	case "minimal":
		return &Config{
			Schema: SchemaURL, Theme: "catppuccin-mocha", Charset: "auto",
			ColorMode: "auto", Separator: " │ ",
			Lines: []Line{
				{Segments: []SegmentConfig{
					{Type: "model"},
					{Type: "context", Options: map[string]any{"display": "percent"}},
					{Type: "cost", Options: map[string]any{"mode": "auto"}},
				}},
				{Segments: []SegmentConfig{
					{Type: "directory", Options: map[string]any{"style": "basename"}},
					{Type: "git", Options: map[string]any{"showSha": false}},
				}},
			},
		}
	case "compact":
		return &Config{
			Schema: SchemaURL, Theme: "catppuccin-mocha", Charset: "auto",
			ColorMode: "auto", Separator: " │ ",
			Lines: []Line{
				{Segments: []SegmentConfig{
					{Type: "model"},
					{Type: "context", Options: map[string]any{"display": "both", "barWidth": 8}},
					{Type: "directory", Options: map[string]any{"style": "basename"}},
					{Type: "git", Options: map[string]any{"showSha": false}},
					{Type: "cost", Options: map[string]any{"mode": "auto"}},
				}},
			},
		}
	}
	return nil
}
