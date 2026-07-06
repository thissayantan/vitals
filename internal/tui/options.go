package tui

import "fmt"

// optionSpec describes one editable option of a segment type.
type optionSpec struct {
	key     string   // key in SegmentConfig.Options
	label   string   // display name
	kind    string   // "enum" | "bool" | "int"
	choices []string // for enum
	def     any      // default (string | bool | int)
}

// segmentOptions lists the editable options per segment type (in display order).
// Segment types not present here have nothing to edit.
var segmentOptions = map[string][]optionSpec{
	"context": {
		{key: "display", label: "display", kind: "enum", choices: []string{"both", "bar", "percent"}, def: "both"},
		{key: "barWidth", label: "bar width", kind: "int", def: 10},
	},
	"block":     {{key: "format", label: "clock", kind: "enum", choices: []string{"12h", "24h"}, def: "12h"}},
	"cost":      {{key: "mode", label: "cost mode", kind: "enum", choices: []string{"auto", "subscription", "api"}, def: "auto"}},
	"directory": {{key: "style", label: "style", kind: "enum", choices: []string{"basename", "full", "truncated"}, def: "basename"}},
	"git":       {{key: "showSha", label: "show sha", kind: "bool", def: false}},
	"tasks":     {{key: "barWidth", label: "bar width", kind: "int", def: 10}},
}

// optionsFor returns the editable options for a segment type (nil if none).
func optionsFor(typ string) []optionSpec { return segmentOptions[typ] }

// optDisplay returns the current value of spec as a display string.
func optDisplay(opts map[string]any, spec optionSpec) string {
	switch spec.kind {
	case "bool":
		return fmt.Sprintf("%v", optBoolVal(opts, spec))
	case "int":
		return fmt.Sprintf("%d", optIntVal(opts, spec))
	default:
		return optStrVal(opts, spec)
	}
}

func optStrVal(opts map[string]any, spec optionSpec) string {
	if v, ok := opts[spec.key].(string); ok {
		return v
	}
	if d, ok := spec.def.(string); ok {
		return d
	}
	return ""
}

func optBoolVal(opts map[string]any, spec optionSpec) bool {
	if v, ok := opts[spec.key].(bool); ok {
		return v
	}
	d, _ := spec.def.(bool)
	return d
}

func optIntVal(opts map[string]any, spec optionSpec) int {
	switch v := opts[spec.key].(type) {
	case int:
		return v
	case float64: // JSON numbers decode to float64
		return int(v)
	}
	d, _ := spec.def.(int)
	return d
}

// cycleOption mutates opts[spec.key] to the next value: enums/ints step by dir
// (+1/-1), bools toggle. opts must be non-nil.
func cycleOption(opts map[string]any, spec optionSpec, dir int) {
	switch spec.kind {
	case "enum":
		opts[spec.key] = cycle(spec.choices, optStrVal(opts, spec), dir)
	case "bool":
		opts[spec.key] = !optBoolVal(opts, spec)
	case "int":
		n := optIntVal(opts, spec) + dir
		if n < 1 {
			n = 1
		}
		if n > 30 {
			n = 30
		}
		opts[spec.key] = n
	}
}
