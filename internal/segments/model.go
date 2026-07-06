package segments

func init() { Register(&modelSegment{}) }

// modelSegment renders the brand marker + model name, e.g. "◆ Opus 4.8".
// Source: model.display_name (context suffix stripped). Never hides.
type modelSegment struct{}

func (s *modelSegment) Type() string { return "model" }

func (s *modelSegment) Render(ctx *RenderCtx, cfg SegmentConfig) (string, bool) {
	glyph := ctx.Theme.Style("brand").Render(ctx.Theme.Glyphs.Brand)
	name := ctx.Session.DisplayModel()

	var nameText string
	if cfg.Style != nil {
		nameText = ctx.Theme.NewStyle(*cfg.Style).Render(name)
	} else {
		ns := ctx.Theme.Style("accent")
		ns.Bold = true
		nameText = ns.Render(name)
	}
	return glyph + " " + nameText, true
}
