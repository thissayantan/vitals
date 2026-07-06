package segments

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thissayantan/vitals/internal/cache"
)

func init() { Register(&tasksSegment{}) }

// tasksSegment renders a task-progress bar + flag, e.g. "▣▣▢░░░░░░░ 30% ⚑".
// Source: ~/.claude/tasks/<session_id>/*.json (count + statuses). Hides when
// there are 0 tasks. Cached 3s by session id.
//
// Options: barWidth (default 10).
type tasksSegment struct{}

func (s *tasksSegment) Type() string { return "tasks" }

type taskCounts struct {
	Total      int `json:"total"`
	Done       int `json:"done"`
	InProgress int `json:"inProgress"`
}

func (s *tasksSegment) Render(ctx *RenderCtx, cfg SegmentConfig) (string, bool) {
	sid := ctx.Session.ResolvedSessionID()
	if sid == "" {
		return "", false
	}
	counts := loadTaskCounts(sid, ctx.Cache)
	if counts.Total <= 0 {
		return "", false
	}

	barWidth := optInt(cfg.Options, "barWidth", 10)
	if barWidth < 1 {
		barWidth = 10
	}

	filled := counts.Done * barWidth / counts.Total
	if filled > barWidth {
		filled = barWidth
	}
	pct := (counts.Done*100 + counts.Total/2) / counts.Total
	showProg := counts.InProgress > 0 && filled < barWidth

	bar := ctx.Theme.TaskBar(filled, barWidth, showProg)

	// Color ramp is the inverse of usage bars: high completion is "ok" (green).
	role := "muted"
	switch {
	case pct >= 80:
		role = "ok"
	case pct >= 40:
		role = "warn"
	}
	pctText := styled(ctx, cfg, role, fmt.Sprintf("%d%%", pct))
	flag := ctx.Theme.Style("muted").Render(ctx.Theme.Glyphs.Flag)

	out := bar + " " + pctText
	if ctx.Theme.Glyphs.Flag != "" {
		out += " " + flag
	}
	return out, true
}

// loadTaskCounts reads ~/.claude/tasks/<sid>/*.json, counting total/done/
// in-progress statuses. Cached 3s.
func loadTaskCounts(sid string, c *cache.Store) taskCounts {
	out := c.Memo("tasks:"+sid, cache.TTLTasks, func() []byte {
		counts := scanTasks(sid)
		b, _ := json.Marshal(counts)
		return b
	})
	var counts taskCounts
	_ = json.Unmarshal(out, &counts)
	return counts
}

func scanTasks(sid string) taskCounts {
	home, err := os.UserHomeDir()
	if err != nil {
		return taskCounts{}
	}
	dir := filepath.Join(home, ".claude", "tasks", sid)
	matches, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil || len(matches) == 0 {
		return taskCounts{}
	}
	var counts taskCounts
	for _, m := range matches {
		data, err := os.ReadFile(m)
		if err != nil {
			continue
		}
		var t struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(data, &t); err != nil {
			continue
		}
		counts.Total++
		switch t.Status {
		case "completed":
			counts.Done++
		case "in_progress":
			counts.InProgress++
		}
	}
	return counts
}
