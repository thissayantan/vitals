package segments

func init() { Register(&gitSegment{}) }

// gitSegment renders the branch + dirty marker, e.g. "⎇ feat/hiring-pipeline*",
// with an optional short SHA. Source: worktree.branch ⇒ .git/HEAD ⇒
// `git rev-parse --short`. Hides when the directory is not a repo.
//
// Options: showSha (bool, default false).
type gitSegment struct{}

func (s *gitSegment) Type() string { return "git" }

func (s *gitSegment) Render(ctx *RenderCtx, cfg SegmentConfig) (string, bool) {
	showSha := optBool(cfg.Options, "showSha", false)

	info := ctx.Git()
	if showSha {
		info = ctx.GitSHA()
	}
	if !info.IsRepo || info.Branch == "" {
		return "", false
	}

	out := styled(ctx, cfg, "git", ctx.Theme.Glyphs.Branch+info.Branch)
	if info.Dirty {
		out += ctx.Theme.Style("warn").Render("*")
	}
	if showSha && info.SHA != "" {
		out += " " + ctx.Theme.Style("muted").Render(info.SHA)
	}
	return out, true
}
