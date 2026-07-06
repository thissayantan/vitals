package segments

import (
	"fmt"
	"strings"
)

func init() { Register(&contextSegment{}) }

// contextSegment renders the context-window badge + gradient bar + percent,
// e.g. "1M ████░░░░░░ 38% ctx".
//
// Source: context_window.{context_window_size,used_percentage}.
// The badge ("1M"/"200k") hides when the window is < 200k; the segment itself
// always shows (the bar/percent are meaningful even at 0%).
//
// Options: display ("bar"|"percent"|"both", default "both"), barWidth (default 10).
type contextSegment struct{}

func (s *contextSegment) Type() string { return "context" }

func (s *contextSegment) Render(ctx *RenderCtx, cfg SegmentConfig) (string, bool) {
	cw := ctx.Session.ContextWindow
	display := optString(cfg.Options, "display", "both")
	barWidth := optInt(cfg.Options, "barWidth", 10)
	if barWidth < 1 {
		barWidth = 10
	}

	pct := int(cw.UsedPercentage)
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	var parts []string

	// Window-size badge (muted), hidden below 200k.
	if badge := windowBadge(cw.ContextWindowSize); badge != "" {
		parts = append(parts, ctx.Theme.Style("muted").Render(badge))
	}

	if display == "bar" || display == "both" {
		filled := pct * barWidth / 100
		parts = append(parts, ctx.Theme.GradientBar(filled, barWidth))
	}

	if display == "percent" || display == "both" {
		pctText := styled(ctx, cfg, pctRole(pct), fmt.Sprintf("%d%%", pct))
		ctxLabel := ctx.Theme.Style("muted").Render("ctx")
		parts = append(parts, pctText+" "+ctxLabel)
	}

	out := strings.Join(parts, " ")
	// Lead with the context (brain) glyph; it carries a trailing space.
	if g := ctx.Theme.Glyphs.Ctx; g != "" {
		out = ctx.Theme.Style("muted").Render(g) + out
	}
	return out, true
}

// windowBadge returns "1M" for ≥1M, "200k" for ≥200k, else "".
func windowBadge(size int64) string {
	switch {
	case size >= 1_000_000:
		return "1M"
	case size >= 200_000:
		return "200k"
	default:
		return ""
	}
}

// pctRole maps a usage percentage to a semantic color role (shared ramp).
func pctRole(pct int) string {
	switch {
	case pct >= 80:
		return "crit"
	case pct >= 60:
		return "warn"
	default:
		return "ok"
	}
}
