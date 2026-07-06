// Package segments defines the Segment interface, the registry, and one file per
// segment module (model.go, context.go, block.go, weekly.go, cost.go,
// duration.go, tasks.go, git.go, diff.go, runtime.go, worktree.go, directory.go).
//
// This is the heart of modularity (DESIGN.md §3–§4). Adding a segment is one new
// file that calls Register in its init() — nothing else in the codebase changes.
package segments

import (
	"sort"

	"github.com/thissayantan/vitals/internal/cache"
	"github.com/thissayantan/vitals/internal/claude"
	"github.com/thissayantan/vitals/internal/cost"
	"github.com/thissayantan/vitals/internal/gitinfo"
	"github.com/thissayantan/vitals/internal/runtime"
	"github.com/thissayantan/vitals/internal/theme"
)

// RenderCtx is everything a segment may need, computed once per invocation and
// shared. The provider funcs are lazily computed and memoized by the renderer,
// so a disabled segment never triggers its cost.
type RenderCtx struct {
	Session *claude.Session
	Theme   *theme.Theme
	Cache   *cache.Store

	Git     func() gitinfo.Info
	GitSHA  func() gitinfo.Info
	Runtime func() runtime.Info
	Cost    func(source string) cost.Estimate
}

// SegmentConfig is the user's per-segment options. Type-specific keys live in
// Options. This is the renderer-facing view of a config.SegmentConfig (the
// config package owns parsing/merging; segments stays decoupled from config).
type SegmentConfig struct {
	Type    string
	Enabled bool
	Style   *theme.Style
	Options map[string]any
}

// Segment renders one piece of the status line.
type Segment interface {
	Type() string
	// Render returns the styled text and whether to show it. show == false ⇒ the
	// segment AND its adjoining separator are omitted entirely.
	Render(ctx *RenderCtx, cfg SegmentConfig) (text string, show bool)
}

// registry holds every segment keyed by Type(). Populated from each module's
// init() via Register.
var registry = map[string]Segment{}

// Register adds a segment to the registry. Called from each segment file's
// init(). Panics on a duplicate type (a programming error caught at startup).
func Register(s Segment) {
	t := s.Type()
	if _, dup := registry[t]; dup {
		panic("segments: duplicate registration for type " + t)
	}
	registry[t] = s
}

// Get returns the registered segment for typ.
func Get(typ string) (Segment, bool) {
	s, ok := registry[typ]
	return s, ok
}

// All returns every registered segment type, sorted. Used by the configurator's
// add-segment picker.
func All() []string {
	out := make([]string, 0, len(registry))
	for t := range registry {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// ─── option helpers (shared by segment modules) ──────────────────────────

// optString reads a string option with a default.
func optString(opts map[string]any, key, def string) string {
	if v, ok := opts[key].(string); ok {
		return v
	}
	return def
}

// optBool reads a bool option with a default.
func optBool(opts map[string]any, key string, def bool) bool {
	if v, ok := opts[key].(bool); ok {
		return v
	}
	return def
}

// optInt reads an int option with a default. JSON numbers decode to float64, so
// accept both.
func optInt(opts map[string]any, key string, def int) int {
	switch v := opts[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return def
}

// styled applies a per-segment style override when present, else the named role.
func styled(ctx *RenderCtx, cfg SegmentConfig, role, text string) string {
	if cfg.Style != nil {
		return ctx.Theme.NewStyle(*cfg.Style).Render(text)
	}
	return ctx.Theme.Style(role).Render(text)
}

// prependGlyph puts a leading muted glyph in front of out, joined by sep (glyphs
// that carry their own trailing space pass sep == ""). Returns out unchanged when
// glyph is empty, i.e. charsets that lack the glyph.
func prependGlyph(ctx *RenderCtx, glyph, sep, out string) string {
	if glyph == "" {
		return out
	}
	return ctx.Theme.Style("muted").Render(glyph) + sep + out
}
