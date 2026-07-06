package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thissayantan/vitals/internal/config"
)

// segIndex returns the flattened cursor index of the first segment of typ.
func segIndex(m model, typ string) int {
	i := 0
	for _, line := range m.cfg.Lines {
		for _, sc := range line.Segments {
			if sc.Type == typ {
				return i
			}
			i++
		}
	}
	return -1
}

func TestOptionEditorCyclesValue(t *testing.T) {
	m := newModel(config.Defaults())
	m.cursor = segIndex(m, "cost")

	updated, _ := m.Update(key("o")) // enter option editor
	m = updated.(model)
	if !m.editing {
		t.Fatal("pressing o should open the option editor")
	}
	updated, _ = m.Update(key("l")) // cycle the (only) option: mode
	m = updated.(model)

	got := m.currentSegment().Options["mode"]
	if got != "subscription" {
		t.Errorf("cost mode = %v, want subscription (auto +1)", got)
	}
	// esc leaves the editor.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(model)
	if m.editing {
		t.Error("esc should close the option editor")
	}
}

func TestRemoveSegment(t *testing.T) {
	m := newModel(config.Defaults())
	m.cursor = 0 // model
	updated, _ := m.Update(key("x"))
	m = updated.(model)
	if segIndex(m, "model") != -1 {
		t.Error("model segment should be removed")
	}
}

func TestAddSegment(t *testing.T) {
	m := newModel(config.Defaults())
	// Remove model, then add it back via the picker.
	m.cursor = 0
	updated, _ := m.Update(key("x"))
	m = updated.(model)

	updated, _ = m.Update(key("a")) // open picker
	m = updated.(model)
	if !m.adding {
		t.Fatal("pressing a should open the add picker")
	}
	// model sorts first among the addable types; select it.
	if m.addableTypes()[0] != "model" {
		t.Fatalf("expected model addable first, got %v", m.addableTypes())
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if m.adding {
		t.Error("enter should close the add picker")
	}
	if segIndex(m, "model") == -1 {
		t.Error("model should be re-added")
	}
}

func TestCyclePreset(t *testing.T) {
	m := newModel(config.Defaults())
	updated, _ := m.Update(key("p")) // full -> minimal
	m = updated.(model)

	want := config.Preset("minimal")
	if len(m.cfg.Lines) != len(want.Lines) {
		t.Fatalf("after preset cycle: %d lines, want %d (minimal)", len(m.cfg.Lines), len(want.Lines))
	}
	if m.cfg.Lines[0].Segments[0].Type != want.Lines[0].Segments[0].Type {
		t.Errorf("preset not applied: first seg %q", m.cfg.Lines[0].Segments[0].Type)
	}
}
