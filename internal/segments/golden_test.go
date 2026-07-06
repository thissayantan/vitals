package segments

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sayantan/vitals/internal/claude"
	"github.com/sayantan/vitals/internal/cost"
	"github.com/sayantan/vitals/internal/gitinfo"
	"github.com/sayantan/vitals/internal/runtime"
	"github.com/sayantan/vitals/internal/theme"
)

var update = flag.Bool("update", false, "update golden files")

func TestMain(m *testing.M) {
	// Pin local time to UTC so the block segment's reset clock is deterministic.
	time.Local = time.UTC
	os.Exit(m.Run())
}

func f64(v float64) *float64 { return &v }

// sampleSession mirrors testdata/session.sample.json.
func sampleSession() *claude.Session {
	return &claude.Session{
		Model:         claude.Model{DisplayName: "Opus 4.8", ID: "claude-opus-4-8"},
		ContextWindow: claude.ContextWindow{UsedPercentage: 38, ContextWindowSize: 1_000_000},
		Cost: claude.Cost{
			TotalCostUSD: 34.13, TotalDurationMs: 2_881_000,
			TotalLinesAdded: 2453, TotalLinesRemoved: 439,
		},
		Workspace: claude.Workspace{CurrentDir: "/home/u/hirex", ProjectDir: "/home/u/hirex"},
		RateLimits: claude.RateLimits{
			FiveHour: claude.Limit{UsedPercentage: f64(65), ResetsAt: 1718784600},
			SevenDay: claude.Limit{UsedPercentage: f64(12), ResetsAt: 1719300000},
		},
		Worktree:  claude.Worktree{Branch: "feat/hiring-pipeline"},
		SessionID: "test-session",
	}
}

// testCtx builds a RenderCtx with a NO_COLOR (plain-text) theme and stubbed
// providers so golden output is deterministic and FS-independent.
func testCtx(t *testing.T, s *claude.Session) *RenderCtx {
	t.Helper()
	th, err := theme.Load("catppuccin-mocha", "unicode", "none", nil)
	if err != nil {
		t.Fatalf("theme load: %v", err)
	}
	return &RenderCtx{
		Session: s,
		Theme:   th,
		Cache:   nil,
		Git:     func() gitinfo.Info { return gitinfo.Info{IsRepo: true, Branch: s.Worktree.Branch, Dirty: true} },
		GitSHA: func() gitinfo.Info {
			return gitinfo.Info{IsRepo: true, Branch: s.Worktree.Branch, Dirty: true, SHA: "a1b2c3d"}
		},
		Runtime: func() runtime.Info { return runtime.Info{Language: "bun", Version: "v1.3.14"} },
		Cost: func(source string) cost.Estimate {
			if source == "estimate" {
				return cost.Estimate{USD: 12.50, Estimated: true}
			}
			return cost.Estimate{USD: s.Cost.TotalCostUSD, Estimated: false}
		},
	}
}

func goldenCase(t *testing.T, name string, cfg SegmentConfig, ctx *RenderCtx) {
	t.Helper()
	seg, ok := Get(cfg.Type)
	if !ok {
		t.Fatalf("no segment %q", cfg.Type)
	}
	text, show := seg.Render(ctx, cfg)
	got := text
	if !show {
		got = "<hidden>"
	}

	path := filepath.Join("testdata", "golden", name+".txt")
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s (run -update): %v", path, err)
	}
	if got != string(want) {
		t.Errorf("%s\n got: %q\nwant: %q", name, got, string(want))
	}
}

func cfg(typ string, opts map[string]any) SegmentConfig {
	return SegmentConfig{Type: typ, Enabled: true, Options: opts}
}

func TestSegmentGoldens(t *testing.T) {
	s := sampleSession()
	ctx := testCtx(t, s)

	cases := []struct {
		name string
		cfg  SegmentConfig
	}{
		{"model", cfg("model", nil)},
		{"context_both", cfg("context", map[string]any{"display": "both", "barWidth": 10})},
		{"context_bar", cfg("context", map[string]any{"display": "bar", "barWidth": 10})},
		{"context_percent", cfg("context", map[string]any{"display": "percent"})},
		{"block", cfg("block", nil)},
		{"block_24h", cfg("block", map[string]any{"format": "24h"})},
		{"weekly", cfg("weekly", nil)},
		{"cost_real", cfg("cost", map[string]any{"source": "cc"})},
		{"cost_estimate", cfg("cost", map[string]any{"source": "estimate"})},
		{"duration", cfg("duration", nil)},
		{"diff", cfg("diff", nil)},
		{"git", cfg("git", nil)},
		{"git_sha", cfg("git", map[string]any{"showSha": true})},
		{"runtime", cfg("runtime", nil)},
		{"directory_basename", cfg("directory", map[string]any{"style": "basename"})},
		{"directory_full", cfg("directory", map[string]any{"style": "full"})},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			goldenCase(t, c.name, c.cfg, ctx)
		})
	}
}

func TestHiddenSegments(t *testing.T) {
	// A session with everything zero/absent ⇒ optional segments hide.
	s := &claude.Session{
		Model:         claude.Model{DisplayName: "Opus 4.8"},
		ContextWindow: claude.ContextWindow{ContextWindowSize: 100_000}, // badge hidden (<200k)
		Workspace:     claude.Workspace{CurrentDir: "/tmp/x"},
	}
	ctx := testCtx(t, s)
	ctx.Git = func() gitinfo.Info { return gitinfo.Info{} }
	ctx.Runtime = func() runtime.Info { return runtime.Info{} }
	ctx.Cost = func(string) cost.Estimate { return cost.Estimate{} }

	hidden := []string{"block", "weekly", "cost", "duration", "diff", "git", "runtime", "worktree"}
	for _, typ := range hidden {
		seg, _ := Get(typ)
		if _, show := seg.Render(ctx, cfg(typ, nil)); show {
			t.Errorf("segment %q should hide on empty session", typ)
		}
	}

	// model & directory never hide.
	for _, typ := range []string{"model", "directory"} {
		seg, _ := Get(typ)
		if _, show := seg.Render(ctx, cfg(typ, nil)); !show {
			t.Errorf("segment %q should always show", typ)
		}
	}
}
