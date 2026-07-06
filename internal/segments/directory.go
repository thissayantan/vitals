package segments

import (
	"os"
	"path/filepath"
	"strings"
)

func init() { Register(&directorySegment{}) }

// directorySegment renders the working directory. Source: workspace.current_dir.
// Never hides.
//
// Options: style ("basename"|"full"|"truncated", default "basename").
type directorySegment struct{}

func (s *directorySegment) Type() string { return "directory" }

func (s *directorySegment) Render(ctx *RenderCtx, cfg SegmentConfig) (string, bool) {
	style := optString(cfg.Options, "style", "basename")
	dir := ctx.Session.CurrentDir()

	var text string
	switch style {
	case "full":
		text = withTilde(dir)
	case "truncated":
		text = truncatePath(withTilde(dir))
	default: // basename
		text = ctx.Session.DirName()
	}
	// Prefix a folder glyph (Nerd Font only; the glyph carries a trailing space).
	if g := ctx.Theme.Glyphs.Dir; g != "" {
		text = g + text
	}
	return styled(ctx, cfg, "dir", text), true
}

// withTilde replaces a leading home directory with "~".
func withTilde(dir string) string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		if dir == home {
			return "~"
		}
		if rest, ok := strings.CutPrefix(dir, home+string(filepath.Separator)); ok {
			return "~" + string(filepath.Separator) + rest
		}
	}
	return dir
}

// truncatePath keeps the last two path components, prefixing "…/" when more were
// elided (e.g. "~/a/b/c/hirex" → "…/c/hirex").
func truncatePath(p string) string {
	sep := string(filepath.Separator)
	parts := strings.Split(p, sep)
	// Drop a leading empty element from an absolute path.
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}
	if len(parts) <= 2 {
		return p
	}
	tail := parts[len(parts)-2:]
	return "…" + sep + strings.Join(tail, sep)
}
