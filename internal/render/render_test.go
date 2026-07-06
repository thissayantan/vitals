package render

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thissayantan/vitals/internal/claude"
	"github.com/thissayantan/vitals/internal/config"
	"github.com/thissayantan/vitals/internal/theme"
)

func TestMain(m *testing.M) {
	time.Local = time.UTC // deterministic reset clock
	os.Exit(m.Run())
}

func f64(v float64) *float64 { return &v }

// TestEndToEnd renders the default two-line layout with a NO_COLOR theme and a
// working directory with no git/lockfiles, so optional FS-driven segments hide
// deterministically.
func TestEndToEnd(t *testing.T) {
	dir := t.TempDir() // empty: not a repo, no lockfile ⇒ git & runtime hide

	s := &claude.Session{
		Model:         claude.Model{DisplayName: "Opus 4.8 (1M context)", ID: "claude-opus-4-8"},
		ContextWindow: claude.ContextWindow{UsedPercentage: 38, ContextWindowSize: 1_000_000},
		Cost: claude.Cost{
			TotalCostUSD: 34.13, TotalDurationMs: 2_881_000,
			TotalLinesAdded: 2453, TotalLinesRemoved: 439,
		},
		Workspace: claude.Workspace{CurrentDir: dir, ProjectDir: dir},
		RateLimits: claude.RateLimits{
			FiveHour: claude.Limit{UsedPercentage: f64(65), ResetsAt: 1718784600},
			SevenDay: claude.Limit{UsedPercentage: f64(12), ResetsAt: 1719300000},
		},
		SessionID: "e2e-none",
	}

	cfg := config.Defaults()
	cfg.ColorMode = "none"
	th, err := theme.Load(cfg.Theme, "unicode", cfg.ColorMode, cfg.ThemeOverrides)
	if err != nil {
		t.Fatal(err)
	}

	got := Render(s, cfg, th)

	want := "◆ Opus 4.8 │ 1M ███░░░░░░░ 38% ctx │ ↻ 5h 65% ↺8:10am │ 7d 12% │ ⏱ 48m1s │ $34.13\n" +
		filepath.Base(dir) + " │ ± +2453/-439"

	if got != want {
		t.Errorf("end-to-end mismatch:\n got: %q\nwant: %q", got, want)
	}
}
