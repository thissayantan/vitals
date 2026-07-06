package segments

func init() { Register(&runtimeSegment{}) }

// runtimeSegment renders the project language + version, e.g. "go v1.24.0"
// (prefixed with a language icon in the Nerd Font charset). Source:
// lockfile/manifest detection (internal/runtime). Hides when no language is
// detected.
type runtimeSegment struct{}

func (s *runtimeSegment) Type() string { return "runtime" }

func (s *runtimeSegment) Render(ctx *RenderCtx, cfg SegmentConfig) (string, bool) {
	info := ctx.Runtime()
	if info.Language == "" {
		return "", false
	}
	label := info.Language
	if icon := ctx.Theme.Glyphs.LangIcon(info.Language); icon != "" {
		label = icon + " " + info.Language
	}
	out := styled(ctx, cfg, "accent", label)
	if info.Version != "" {
		out += " " + ctx.Theme.Style("muted").Render(info.Version)
	}
	return out, true
}
