package theme

import "strings"

// GradientBar renders a `total`-cell bar where the first `filled` cells use the
// theme gradient (position-mapped, green→red) and the rest are dim empty cells.
// In ModeNone it degrades to plain filled/empty glyphs.
func (t *Theme) GradientBar(filled, total int) string {
	if total <= 0 {
		return ""
	}
	if filled < 0 {
		filled = 0
	}
	if filled > total {
		filled = total
	}
	var b strings.Builder
	for i := 0; i < total; i++ {
		if i < filled {
			gi := i
			if len(t.Gradient) > 0 {
				gi = i * len(t.Gradient) / total
			}
			b.WriteString(t.GradientColor(gi).Render(t.Glyphs.BarFilled))
		} else {
			b.WriteString(t.EmptyCell().Render(t.Glyphs.BarEmpty))
		}
	}
	return b.String()
}

// TaskBar renders the task-progress bar: `filled` completed cells (ok color),
// optionally one in-progress cell (warn color) right after them, and the rest
// dim empty cells. Uses the task glyphs (▣ / ▢).
func (t *Theme) TaskBar(filled, total int, showInProgress bool) string {
	if total <= 0 {
		return ""
	}
	if filled < 0 {
		filled = 0
	}
	if filled > total {
		filled = total
	}
	ok := t.Style("ok")
	warn := t.Style("warn")
	empty := t.EmptyCell()
	var b strings.Builder
	for i := 0; i < total; i++ {
		switch {
		case i < filled:
			b.WriteString(ok.Render(t.Glyphs.TaskFilled))
		case i == filled && showInProgress:
			b.WriteString(warn.Render(t.Glyphs.TaskFilled))
		default:
			b.WriteString(empty.Render(t.Glyphs.TaskEmpty))
		}
	}
	return b.String()
}
