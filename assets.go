// Package vitals is the module root. It exists only to embed shared data assets
// (theme palettes, the config JSON Schema, the cost pricing table) so that
// internal packages can read them via a single go:embed FS. go:embed cannot
// reference parent directories, so the embed must live at the module root where
// themes/ and schema/ are visible.
package vitals

import "embed"

// Themes holds the built-in theme palettes (themes/*.json).
//
//go:embed themes/*.json
var Themes embed.FS

// Schema holds the config JSON Schema used for validation.
//
//go:embed schema/vitals.schema.json
var Schema embed.FS
