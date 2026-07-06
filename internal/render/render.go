// Package render composes segments into the final multi-line status output.
//
// See DESIGN.md §3. It iterates config.Lines → renders enabled segments via the
// registry, drops hidden ones, and joins with the separator (no dangling
// separators). The shared RenderCtx is built once, wiring lazily-memoized
// providers so expensive work happens at most once per invocation. The TUI live
// preview reuses Render so the preview is exact.
package render

import (
	"strings"
	"sync"
	"unicode"

	"github.com/sayantan/vitals/internal/cache"
	"github.com/sayantan/vitals/internal/claude"
	"github.com/sayantan/vitals/internal/config"
	"github.com/sayantan/vitals/internal/cost"
	"github.com/sayantan/vitals/internal/gitinfo"
	"github.com/sayantan/vitals/internal/runtime"
	"github.com/sayantan/vitals/internal/segments"
	"github.com/sayantan/vitals/internal/theme"
)

// Render composes the full status output (one string, lines separated by "\n").
// Lines that produce no visible segments are dropped.
func Render(s *claude.Session, cfg *config.Config, th *theme.Theme) string {
	ctx := newCtx(s, th)
	sep := effectiveSeparator(cfg.Separator, th.Charset)

	var lines []string
	for _, line := range cfg.Lines {
		rendered := renderLine(ctx, line, sep)
		if rendered != "" {
			lines = append(lines, rendered)
		}
	}
	return strings.Join(lines, "\n")
}

// effectiveSeparator falls the separator back to " | " when the ASCII charset is
// active but the configured separator contains non-ASCII glyphs (e.g. the
// default "│"), preserving the prototype's pure-ASCII guarantee for old terminals.
func effectiveSeparator(sep string, cs theme.Charset) string {
	if cs != theme.CharsetASCII {
		return sep
	}
	for _, r := range sep {
		if r > unicode.MaxASCII {
			return " | "
		}
	}
	return sep
}

// renderLine renders one line's enabled, visible segments joined by sep.
func renderLine(ctx *segments.RenderCtx, line config.Line, sep string) string {
	var parts []string
	for _, sc := range line.Segments {
		if !sc.IsEnabled() {
			continue
		}
		seg, ok := segments.Get(sc.Type)
		if !ok {
			continue
		}
		text, show := seg.Render(ctx, toSegmentConfig(sc))
		if !show || text == "" {
			continue
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, sep)
}

// toSegmentConfig adapts a config.SegmentConfig to the segments-package view.
func toSegmentConfig(sc config.SegmentConfig) segments.SegmentConfig {
	return segments.SegmentConfig{
		Type:    sc.Type,
		Enabled: sc.IsEnabled(),
		Style:   sc.Style,
		Options: sc.Options,
	}
}

// newCtx builds the shared RenderCtx, wiring each expensive provider behind a
// sync.Once so it runs at most once and only when a segment actually calls it.
func newCtx(s *claude.Session, th *theme.Theme) *segments.RenderCtx {
	store := cache.New(s.ResolvedSessionID())

	var (
		gitOnce sync.Once
		gitInfo gitinfo.Info

		gitSHAOnce sync.Once
		gitSHAInfo gitinfo.Info

		rtOnce sync.Once
		rtInfo runtime.Info
	)

	return &segments.RenderCtx{
		Session: s,
		Theme:   th,
		Cache:   store,
		Git: func() gitinfo.Info {
			gitOnce.Do(func() {
				gitInfo = gitinfo.Get(s.CurrentDir(), s.Worktree.Branch, store)
			})
			return gitInfo
		},
		GitSHA: func() gitinfo.Info {
			gitSHAOnce.Do(func() {
				gitSHAInfo = gitinfo.GetSHA(s.CurrentDir(), s.Worktree.Branch, store)
			})
			return gitSHAInfo
		},
		Runtime: func() runtime.Info {
			rtOnce.Do(func() {
				rtInfo = runtime.Get(s.ProjectDir(), store)
			})
			return rtInfo
		},
		Cost: func(source string) cost.Estimate {
			// Cost depends on the source option, so memoize per source.
			return cost.Get(s, store, source)
		},
	}
}
