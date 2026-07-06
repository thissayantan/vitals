package segments

import "fmt"

func init() { Register(&durationSegment{}) }

// durationSegment renders the session wall-clock duration, e.g. "⏱ 48m1s" or
// "⏱ 5h 23m". Source: cost.total_duration_ms. Hides when < 1000ms.
type durationSegment struct{}

func (s *durationSegment) Type() string { return "duration" }

func (s *durationSegment) Render(ctx *RenderCtx, cfg SegmentConfig) (string, bool) {
	ms := ctx.Session.Cost.TotalDurationMs
	if ms < 1000 {
		return "", false
	}
	sec := ms / 1000
	h := sec / 3600
	m := (sec % 3600) / 60
	s2 := sec % 60

	var text string
	switch {
	case h > 0:
		text = fmt.Sprintf("%dh %dm", h, m)
	case m > 0:
		text = fmt.Sprintf("%dm%ds", m, s2)
	default:
		text = fmt.Sprintf("%ds", s2)
	}

	out := ctx.Theme.Glyphs.Clock + text
	return styled(ctx, cfg, "muted", out), true
}
