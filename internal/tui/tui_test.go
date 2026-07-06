package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thissayantan/vitals/internal/config"
)

func key(s string) tea.KeyMsg {
	switch s {
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func TestToggleDisablesSegment(t *testing.T) {
	m := newModel(config.Defaults())
	// Cursor starts at the first segment (model). Toggle it off.
	updated, _ := m.Update(key(" "))
	m = updated.(model)

	first := m.cfg.Lines[0].Segments[0]
	if first.IsEnabled() {
		t.Error("first segment should be disabled after toggle")
	}
}

func TestReorderAcrossLines(t *testing.T) {
	m := newModel(config.Defaults())
	// Move the first segment of line 1 (model) up to... it's already at top of
	// line 1; moving up does nothing. Instead move cursor to first of line 2 and
	// move it up into line 1.
	firstLine2 := len(m.cfg.Lines[0].Segments) // flattened index of line2[0]
	m.cursor = firstLine2
	wantType := m.cfg.Lines[1].Segments[0].Type

	updated, _ := m.Update(key("K")) // move up → end of line 1
	m = updated.(model)

	last1 := m.cfg.Lines[0].Segments[len(m.cfg.Lines[0].Segments)-1]
	if last1.Type != wantType {
		t.Errorf("segment %q should have moved to end of line 1, got %q", wantType, last1.Type)
	}
}

func TestReorderWithinLine(t *testing.T) {
	m := newModel(config.Defaults())
	a := m.cfg.Lines[0].Segments[0].Type
	b := m.cfg.Lines[0].Segments[1].Type

	m.cursor = 0
	updated, _ := m.Update(key("J")) // move first segment down
	m = updated.(model)

	if m.cfg.Lines[0].Segments[0].Type != b || m.cfg.Lines[0].Segments[1].Type != a {
		t.Errorf("expected swap of first two segments, got %q,%q",
			m.cfg.Lines[0].Segments[0].Type, m.cfg.Lines[0].Segments[1].Type)
	}
	// Cursor follows the moved segment.
	if m.flat()[m.cursor] != (cursorPos{0, 1}) {
		t.Errorf("cursor should follow moved segment to (0,1), got %+v", m.flat()[m.cursor])
	}
}

func TestThemeCycle(t *testing.T) {
	m := newModel(config.Defaults())
	start := m.cfg.Theme
	updated, _ := m.Update(key("t"))
	m = updated.(model)
	if m.cfg.Theme == start {
		t.Error("theme should change on 't'")
	}
}

func TestViewRenders(t *testing.T) {
	m := newModel(config.Defaults())
	out := m.View()
	if !strings.Contains(out, "vitals config") {
		t.Error("view should contain title")
	}
	if !strings.Contains(out, "live preview") {
		t.Error("view should contain the preview label")
	}
	if !strings.Contains(out, "Line 1") || !strings.Contains(out, "Line 2") {
		t.Error("view should list both lines")
	}
}

func TestCycleWrap(t *testing.T) {
	if got := cycle([]string{"a", "b", "c"}, "c", +1); got != "a" {
		t.Errorf("cycle wrap forward = %q, want a", got)
	}
	if got := cycle([]string{"a", "b", "c"}, "a", -1); got != "c" {
		t.Errorf("cycle wrap backward = %q, want c", got)
	}
	if got := cycle([]string{"a", "b"}, "absent", +1); got != "a" {
		t.Errorf("cycle absent = %q, want a", got)
	}
}
