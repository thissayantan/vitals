package segments

func init() { Register(&runtimeSegment{}) }

// runtimeSegment renders the project language + version, e.g. "bun v1.3.14".
// Source: lockfile/manifest detection (internal/runtime). Hides when no
// language is detected.
type runtimeSegment struct{}

func (s *runtimeSegment) Type() string { return "runtime" }

func (s *runtimeSegment) Render(ctx *RenderCtx, cfg SegmentConfig) (string, bool) {
	info := ctx.Runtime()
	if info.Language == "" {
		return "", false
	}
	out := styled(ctx, cfg, "accent", info.Language)
	if info.Version != "" {
		out += " " + ctx.Theme.Style("muted").Render(info.Version)
	}
	return out, true
}
