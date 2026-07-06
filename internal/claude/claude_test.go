package claude

import (
	"os"
	"strings"
	"testing"
)

func TestParseSample(t *testing.T) {
	f, err := os.Open("../../testdata/session.sample.json")
	if err != nil {
		t.Fatalf("open sample: %v", err)
	}
	defer func() { _ = f.Close() }()

	s, err := Parse(f)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if got := s.DisplayModel(); got != "Opus 4.8" {
		t.Errorf("DisplayModel = %q, want %q", got, "Opus 4.8")
	}
	if s.ContextWindow.UsedPercentage != 38 {
		t.Errorf("ctx used = %v, want 38", s.ContextWindow.UsedPercentage)
	}
	if s.ContextWindow.ContextWindowSize != 1000000 {
		t.Errorf("ctx size = %v, want 1000000", s.ContextWindow.ContextWindowSize)
	}
	if s.Cost.TotalCostUSD != 34.13 {
		t.Errorf("cost = %v, want 34.13", s.Cost.TotalCostUSD)
	}
	if s.RateLimits.FiveHour.UsedPercentage == nil || *s.RateLimits.FiveHour.UsedPercentage != 65 {
		t.Errorf("five_hour pct = %v, want 65", s.RateLimits.FiveHour.UsedPercentage)
	}
	if s.RateLimits.SevenDay.UsedPercentage == nil || *s.RateLimits.SevenDay.UsedPercentage != 12 {
		t.Errorf("seven_day pct = %v, want 12", s.RateLimits.SevenDay.UsedPercentage)
	}
	if s.Worktree.Branch != "feat/hiring-pipeline" {
		t.Errorf("branch = %q", s.Worktree.Branch)
	}
	if s.DirName() != "hirex" {
		t.Errorf("DirName = %q, want hirex", s.DirName())
	}
}

func TestParseEmpty(t *testing.T) {
	s, err := Parse(strings.NewReader(""))
	if err != nil {
		t.Fatalf("parse empty: %v", err)
	}
	if s.DisplayModel() != "Claude" {
		t.Errorf("empty DisplayModel = %q, want Claude", s.DisplayModel())
	}
	if s.RateLimits.FiveHour.UsedPercentage != nil {
		t.Errorf("absent five_hour should be nil")
	}
}

func TestStripContextSuffix(t *testing.T) {
	s := &Session{Model: Model{DisplayName: "Opus 4.8 (1M context)"}}
	if got := s.DisplayModel(); got != "Opus 4.8" {
		t.Errorf("DisplayModel = %q, want %q", got, "Opus 4.8")
	}
}
