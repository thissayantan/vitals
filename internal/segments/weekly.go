package segments

import "fmt"

func init() { Register(&weeklySegment{}) }

// weeklySegment renders the 7-day rate-limit usage, e.g. "7d 12%".
//
// Source: rate_limits.seven_day.used_percentage. Hides when the value is absent
// or ≤ 0 (the prototype only shows weekly usage once it is positive).
//
// Color ramp differs from block/context: low usage is muted (not "ok" green).
type weeklySegment struct{}

func (s *weeklySegment) Type() string { return "weekly" }

func (s *weeklySegment) Render(ctx *RenderCtx, cfg SegmentConfig) (string, bool) {
	lim := ctx.Session.RateLimits.SevenDay
	if lim.UsedPercentage == nil {
		return "", false
	}
	pct := int(*lim.UsedPercentage)
	if pct <= 0 {
		return "", false
	}

	role := "muted"
	switch {
	case pct >= 80:
		role = "crit"
	case pct >= 60:
		role = "warn"
	}
	label := ctx.Theme.Style("muted").Render("7d")
	value := styled(ctx, cfg, role, fmt.Sprintf("%d%%", pct))
	// Lead with the weekly (calendar) glyph; it carries a trailing space.
	out := prependGlyph(ctx, ctx.Theme.Glyphs.Week, "", label+" "+value)
	return out, true
}
