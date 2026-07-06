package segments

import "fmt"

func init() { Register(&diffSegment{}) }

// diffSegment renders lines added/removed, e.g. "+2453/-439" (green/red).
// Source: cost.total_lines_{added,removed}. Hides when both are 0.
type diffSegment struct{}

func (s *diffSegment) Type() string { return "diff" }

func (s *diffSegment) Render(ctx *RenderCtx, _ SegmentConfig) (string, bool) {
	add := ctx.Session.Cost.TotalLinesAdded
	rm := ctx.Session.Cost.TotalLinesRemoved
	if add <= 0 && rm <= 0 {
		return "", false
	}
	addText := ctx.Theme.Style("add").Render(fmt.Sprintf("+%d", add))
	sep := ctx.Theme.Style("muted").Render("/")
	rmText := ctx.Theme.Style("del").Render(fmt.Sprintf("-%d", rm))
	out := prependGlyph(ctx, ctx.Theme.Glyphs.Diff, " ", addText+sep+rmText)
	return out, true
}
