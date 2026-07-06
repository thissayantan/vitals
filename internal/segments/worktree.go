package segments

func init() { Register(&worktreeSegment{}) }

// worktreeSegment renders the active worktree name, e.g. "⚙ wt:name".
// Source: worktree.name. Hides when empty.
type worktreeSegment struct{}

func (s *worktreeSegment) Type() string { return "worktree" }

func (s *worktreeSegment) Render(ctx *RenderCtx, cfg SegmentConfig) (string, bool) {
	name := ctx.Session.Worktree.Name
	if name == "" {
		return "", false
	}
	return styled(ctx, cfg, "warn", ctx.Theme.Glyphs.Gear+"wt:"+name), true
}
