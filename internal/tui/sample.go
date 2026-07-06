package tui

import (
	"os"

	"github.com/sayantan/vitals/internal/claude"
)

// previewSession returns a representative session for the live preview. The
// numeric fields mirror testdata/session.sample.json, but the working directory
// is the real cwd so the git/runtime/directory segments reflect the user's
// actual environment ("the real last session if available", DESIGN.md §9).
func previewSession() *claude.Session {
	f := func(v float64) *float64 { return &v }

	dir, err := os.Getwd()
	if err != nil || dir == "" {
		dir = "/home/u/hirex"
	}

	return &claude.Session{
		Model:         claude.Model{DisplayName: "Opus 4.8", ID: "claude-opus-4-8"},
		ContextWindow: claude.ContextWindow{UsedPercentage: 38, ContextWindowSize: 1_000_000},
		Cost: claude.Cost{
			TotalCostUSD: 34.13, TotalDurationMs: 2_881_000,
			TotalLinesAdded: 2453, TotalLinesRemoved: 439,
		},
		Workspace: claude.Workspace{CurrentDir: dir, ProjectDir: dir},
		RateLimits: claude.RateLimits{
			FiveHour: claude.Limit{UsedPercentage: f(65), ResetsAt: 1718784600},
			SevenDay: claude.Limit{UsedPercentage: f(12), ResetsAt: 1719300000},
		},
		SessionID: "preview",
	}
}
