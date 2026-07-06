package render

import (
	"testing"
	"time"

	"github.com/thissayantan/vitals/internal/claude"
	"github.com/thissayantan/vitals/internal/config"
	"github.com/thissayantan/vitals/internal/theme"
)

// TestRenderLatency guards the hot path: a single render must stay well under
// Claude Code's 300ms debounce budget (DESIGN.md §3). We assert a conservative
// 50ms ceiling on the warm path with FS-driven segments pointing at an empty
// temp dir.
func TestRenderLatency(t *testing.T) {
	dir := t.TempDir()
	s := &claude.Session{
		Model:         claude.Model{DisplayName: "Opus 4.8"},
		ContextWindow: claude.ContextWindow{UsedPercentage: 38, ContextWindowSize: 1_000_000},
		Cost:          claude.Cost{TotalCostUSD: 34.13, TotalDurationMs: 2_881_000, TotalLinesAdded: 10, TotalLinesRemoved: 2},
		Workspace:     claude.Workspace{CurrentDir: dir, ProjectDir: dir},
		RateLimits: claude.RateLimits{
			FiveHour: claude.Limit{UsedPercentage: f64(65), ResetsAt: 1718784600},
		},
		SessionID: "latency",
	}
	cfg := config.Defaults()
	th, err := theme.Load(cfg.Theme, "unicode", "truecolor", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Warm caches once, then measure.
	_ = Render(s, cfg, th)

	start := time.Now()
	const n = 20
	for i := 0; i < n; i++ {
		_ = Render(s, cfg, th)
	}
	per := time.Since(start) / n
	t.Logf("avg render: %v", per)
	if per > 50*time.Millisecond {
		t.Errorf("render too slow: %v > 50ms", per)
	}
}
