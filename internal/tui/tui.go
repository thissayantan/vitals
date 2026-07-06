// Package tui implements `vitals config`, the in-terminal configurator.
//
// See DESIGN.md §9. Built with charmbracelet bubbletea (MVU loop) + lipgloss
// (styling). The LIVE PREVIEW pane is rendered with internal/render +
// internal/theme — the exact same code as the real renderer — so what you see
// is what you get. Save writes ~/.config/vitals/config.json.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/thissayantan/vitals/internal/claude"
	"github.com/thissayantan/vitals/internal/config"
	"github.com/thissayantan/vitals/internal/render"
	"github.com/thissayantan/vitals/internal/segments"
	"github.com/thissayantan/vitals/internal/theme"
)

// themeChoices are the built-in themes cycled in the configurator.
var themeChoices = []string{"catppuccin-mocha", "nord", "tokyo-night", "gruvbox", "rose-pine", "none"}

// charsetChoices are the glyph sets cycled in the configurator.
var charsetChoices = []string{"auto", "unicode", "nerdfont", "ascii"}

// separatorChoices are the separator presets cycled in the configurator.
var separatorChoices = []string{" │ ", " • ", "  ", " | ", " · ", " ┃ "}

// Run loads the config, runs the configurator, and (on save) writes it out.
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.Defaults()
	}
	p := tea.NewProgram(newModel(cfg), tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// cursorPos identifies a segment by its (line, index) position.
type cursorPos struct{ line, seg int }

type model struct {
	cfg     *config.Config
	cursor  int // index into the flattened segment list
	width   int
	height  int
	status  string
	saved   bool
	preview *claude.Session

	editing   bool // per-segment option editor is open
	optCursor int  // cursor within the current segment's options
	adding    bool // add-segment type picker is open
	addCursor int  // cursor within the addable-types list
	presetIdx int  // last-applied preset index (for cycling)
}

const listHelp = "↑/↓ move · space toggle · o options · a add · x remove · J/K reorder · p preset · t/c/[/] theme/charset/sep · s save · q quit"

func newModel(cfg *config.Config) model {
	return model{
		cfg:     cfg,
		preview: previewSession(),
		status:  listHelp,
	}
}

func (m model) Init() tea.Cmd { return nil }

// flat returns the navigable list of segment positions in display order.
func (m model) flat() []cursorPos {
	var out []cursorPos
	for li, line := range m.cfg.Lines {
		for si := range line.Segments {
			out = append(out, cursorPos{li, si})
		}
	}
	return out
}

func (m *model) clampCursor() {
	n := len(m.flat())
	switch {
	case n == 0:
		m.cursor = 0
	case m.cursor < 0:
		m.cursor = 0
	case m.cursor >= n:
		m.cursor = n - 1
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case m.editing:
			m.updateEditing(msg)
		case m.adding:
			m.updateAdding(msg)
		default:
			return m.updateList(msg)
		}
	}
	return m, nil
}

// updateList handles keys in the main segment-list view.
func (m model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit

	case "up", "k":
		m.cursor--
		m.clampCursor()
	case "down", "j":
		m.cursor++
		m.clampCursor()

	case " ":
		m.toggleCurrent()
	case "o", "enter":
		m.openOptions()
	case "a":
		m.openAdd()
	case "x", "d":
		m.removeCurrent()
	case "J":
		m.moveCurrent(+1)
	case "K":
		m.moveCurrent(-1)
	case "p":
		m.cyclePreset()

	case "t":
		m.cfg.Theme = cycle(themeChoices, m.cfg.Theme, +1)
		m.status = "theme: " + m.cfg.Theme
	case "c":
		m.cfg.Charset = cycle(charsetChoices, m.cfg.Charset, +1)
		m.status = "charset: " + m.cfg.Charset
	case "]":
		m.cfg.Separator = cycle(separatorChoices, m.cfg.Separator, +1)
	case "[":
		m.cfg.Separator = cycle(separatorChoices, m.cfg.Separator, -1)

	case "s":
		m.save()
	}
	return m, nil
}

// updateEditing handles keys in the per-segment option editor.
func (m *model) updateEditing(msg tea.KeyMsg) {
	specs := optionsFor(m.currentType())
	switch msg.String() {
	case "esc", "o", "q":
		m.editing = false
		m.status = listHelp
	case "up", "k":
		if m.optCursor > 0 {
			m.optCursor--
		}
	case "down", "j":
		if m.optCursor < len(specs)-1 {
			m.optCursor++
		}
	case "left", "h":
		m.editOption(-1)
	case "right", "l", " ", "enter":
		m.editOption(+1)
	}
}

// updateAdding handles keys in the add-segment type picker.
func (m *model) updateAdding(msg tea.KeyMsg) {
	types := m.addableTypes()
	switch msg.String() {
	case "esc", "q":
		m.adding = false
		m.status = listHelp
	case "up", "k":
		if m.addCursor > 0 {
			m.addCursor--
		}
	case "down", "j":
		if m.addCursor < len(types)-1 {
			m.addCursor++
		}
	case "enter", " ":
		if m.addCursor < len(types) {
			m.addSegment(types[m.addCursor])
		}
		m.adding = false
		m.status = listHelp
	}
}

func (m *model) toggleCurrent() {
	flat := m.flat()
	if m.cursor >= len(flat) {
		return
	}
	pos := flat[m.cursor]
	sc := &m.cfg.Lines[pos.line].Segments[pos.seg]
	v := !sc.IsEnabled()
	sc.Enabled = &v
}

// moveCurrent reorders the segment under the cursor by dir (+1 down, -1 up),
// crossing line boundaries (down past a line's end ⇒ start of the next line).
func (m *model) moveCurrent(dir int) {
	flat := m.flat()
	if m.cursor >= len(flat) {
		return
	}
	pos := flat[m.cursor]
	var newPos cursorPos
	if dir > 0 {
		newPos = m.moveDown(pos)
	} else {
		newPos = m.moveUp(pos)
	}
	for i, p := range m.flat() {
		if p == newPos {
			m.cursor = i
			break
		}
	}
}

func (m *model) moveDown(pos cursorPos) cursorPos {
	line := &m.cfg.Lines[pos.line]
	if pos.seg < len(line.Segments)-1 {
		segs := line.Segments
		segs[pos.seg], segs[pos.seg+1] = segs[pos.seg+1], segs[pos.seg]
		return cursorPos{pos.line, pos.seg + 1}
	}
	if pos.line < len(m.cfg.Lines)-1 {
		sc := line.Segments[pos.seg]
		line.Segments = removeAt(line.Segments, pos.seg)
		next := &m.cfg.Lines[pos.line+1]
		next.Segments = insertAt(next.Segments, 0, sc)
		return cursorPos{pos.line + 1, 0}
	}
	return pos
}

func (m *model) moveUp(pos cursorPos) cursorPos {
	line := &m.cfg.Lines[pos.line]
	if pos.seg > 0 {
		segs := line.Segments
		segs[pos.seg], segs[pos.seg-1] = segs[pos.seg-1], segs[pos.seg]
		return cursorPos{pos.line, pos.seg - 1}
	}
	if pos.line > 0 {
		sc := line.Segments[pos.seg]
		line.Segments = removeAt(line.Segments, pos.seg)
		prev := &m.cfg.Lines[pos.line-1]
		prev.Segments = append(prev.Segments, sc)
		return cursorPos{pos.line - 1, len(prev.Segments) - 1}
	}
	return pos
}

func (m *model) save() {
	path := config.UserConfigPath()
	if err := m.cfg.Save(path); err != nil {
		m.status = "save failed: " + err.Error()
		return
	}
	m.saved = true
	m.status = "saved to " + path + "  ·  run `vitals init` to wire it up"
}

// currentType returns the segment type under the cursor ("" if none).
func (m model) currentType() string {
	if sc := m.currentSegment(); sc != nil {
		return sc.Type
	}
	return ""
}

// currentSegment returns a pointer to the segment under the cursor (nil if none).
func (m *model) currentSegment() *config.SegmentConfig {
	flat := m.flat()
	if m.cursor < 0 || m.cursor >= len(flat) {
		return nil
	}
	p := flat[m.cursor]
	return &m.cfg.Lines[p.line].Segments[p.seg]
}

// openOptions enters the per-segment option editor (no-op if the segment has none).
func (m *model) openOptions() {
	if len(optionsFor(m.currentType())) == 0 {
		m.status = "no editable options for " + m.currentType()
		return
	}
	m.editing = true
	m.optCursor = 0
	m.status = "↑/↓ option · ←/→ change · esc back"
}

// editOption cycles the option under optCursor by dir.
func (m *model) editOption(dir int) {
	sc := m.currentSegment()
	if sc == nil {
		return
	}
	specs := optionsFor(sc.Type)
	if m.optCursor < 0 || m.optCursor >= len(specs) {
		return
	}
	if sc.Options == nil {
		sc.Options = map[string]any{}
	}
	cycleOption(sc.Options, specs[m.optCursor], dir)
}

// openAdd enters the add-segment type picker (no-op if every type is present).
func (m *model) openAdd() {
	if len(m.addableTypes()) == 0 {
		m.status = "all segment types already present"
		return
	}
	m.adding = true
	m.addCursor = 0
	m.status = "↑/↓ type · enter add · esc cancel"
}

// addableTypes lists registered segment types not already in the layout, sorted.
func (m model) addableTypes() []string {
	present := map[string]bool{}
	for _, line := range m.cfg.Lines {
		for _, sc := range line.Segments {
			present[sc.Type] = true
		}
	}
	var out []string
	for _, t := range segments.All() {
		if !present[t] {
			out = append(out, t)
		}
	}
	return out
}

// addSegment inserts typ right after the cursor (or on line 1 if the list is empty).
func (m *model) addSegment(typ string) {
	flat := m.flat()
	sc := config.SegmentConfig{Type: typ}
	if m.cursor < len(flat) {
		p := flat[m.cursor]
		line := &m.cfg.Lines[p.line]
		line.Segments = insertAt(line.Segments, p.seg+1, sc)
		m.cursor++
	} else if len(m.cfg.Lines) > 0 {
		m.cfg.Lines[0].Segments = append(m.cfg.Lines[0].Segments, sc)
	}
	m.clampCursor()
	m.status = "added " + typ
}

// removeCurrent deletes the segment under the cursor.
func (m *model) removeCurrent() {
	flat := m.flat()
	if m.cursor >= len(flat) {
		return
	}
	p := flat[m.cursor]
	line := &m.cfg.Lines[p.line]
	typ := line.Segments[p.seg].Type
	line.Segments = removeAt(line.Segments, p.seg)
	m.clampCursor()
	m.status = "removed " + typ
}

// cyclePreset replaces the layout with the next named preset (keeps theme/charset/
// separator so the user's look is preserved).
func (m *model) cyclePreset() {
	names := config.PresetNames()
	m.presetIdx = (m.presetIdx + 1) % len(names)
	p := config.Preset(names[m.presetIdx])
	if p == nil {
		return
	}
	m.cfg.Lines = p.Lines
	m.cursor = 0
	m.clampCursor()
	m.status = "preset: " + names[m.presetIdx]
}

func (m model) View() string {
	var b strings.Builder

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#cba6f7"))
	b.WriteString(title.Render("vitals config"))
	b.WriteString("\n\n")

	b.WriteString(m.renderPreview())
	b.WriteString("\n\n")

	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086"))
	b.WriteString(dim.Render(fmt.Sprintf("theme: %s   charset: %s   separator: %q",
		m.cfg.Theme, m.cfg.Charset, m.cfg.Separator)))
	b.WriteString("\n\n")

	b.WriteString(m.renderList())
	switch {
	case m.editing:
		b.WriteString("\n")
		b.WriteString(m.renderOptions())
	case m.adding:
		b.WriteString("\n")
		b.WriteString(m.renderAdd())
	}
	b.WriteString("\n")
	b.WriteString(dim.Render(m.status))
	b.WriteString("\n")
	return b.String()
}

// renderOptions draws the per-segment option editor for the current segment.
func (m model) renderOptions() string {
	specs := optionsFor(m.currentType())
	header := lipgloss.NewStyle().Foreground(lipgloss.Color("#f9e2af")).Bold(true)
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#89b4fa")).Bold(true)
	val := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1"))

	sc := m.currentSegment()
	var b strings.Builder
	b.WriteString(header.Render("options · " + m.currentType()))
	b.WriteString("\n")
	for i, spec := range specs {
		pointer := "  "
		if i == m.optCursor {
			pointer = cursorStyle.Render("▸ ")
		}
		cur := ""
		if sc != nil {
			cur = optDisplay(sc.Options, spec)
		}
		fmt.Fprintf(&b, "  %s%-10s %s\n", pointer, spec.label, val.Render("‹ "+cur+" ›"))
	}
	return b.String()
}

// renderAdd draws the add-segment type picker.
func (m model) renderAdd() string {
	types := m.addableTypes()
	header := lipgloss.NewStyle().Foreground(lipgloss.Color("#f9e2af")).Bold(true)
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#89b4fa")).Bold(true)

	var b strings.Builder
	b.WriteString(header.Render("add segment"))
	b.WriteString("\n")
	for i, t := range types {
		pointer := "  "
		if i == m.addCursor {
			pointer = cursorStyle.Render("▸ ")
		}
		fmt.Fprintf(&b, "  %s%s\n", pointer, t)
	}
	return b.String()
}

func (m model) renderPreview() string {
	th, err := theme.Load(m.cfg.Theme, m.cfg.Charset, previewColorMode(m.cfg.ColorMode), m.cfg.ThemeOverrides)
	if err != nil {
		return "preview unavailable: " + err.Error()
	}
	out := render.Render(m.preview, m.cfg, th)
	if out == "" {
		out = lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")).Render("(all segments hidden)")
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#45475a")).
		Padding(0, 1)
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")).Render("live preview")
	return label + "\n" + box.Render(out)
}

func (m model) renderList() string {
	flat := m.flat()
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#89b4fa")).Bold(true)
	onStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1"))
	offStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")).Strikethrough(true)
	header := lipgloss.NewStyle().Foreground(lipgloss.Color("#f9e2af")).Bold(true)

	var b strings.Builder
	idx := 0
	for li, line := range m.cfg.Lines {
		b.WriteString(header.Render(fmt.Sprintf("Line %d", li+1)))
		b.WriteString("\n")
		for si, sc := range line.Segments {
			isCursor := idx < len(flat) && flat[idx] == cursorPos{li, si}
			pointer := "  "
			if isCursor {
				pointer = cursorStyle.Render("▸ ")
			}
			box := "[ ]"
			name := sc.Type
			if sc.IsEnabled() {
				box = "[✓]"
				name = onStyle.Render(name)
			} else {
				name = offStyle.Render(name)
			}
			fmt.Fprintf(&b, "  %s%s %s\n", pointer, box, name)
			idx++
		}
	}
	return b.String()
}

// previewColorMode keeps an explicit "none" but upgrades "auto" to truecolor so
// the preview is representative of a modern terminal.
func previewColorMode(mode string) string {
	switch mode {
	case "none":
		return "none"
	case "", "auto":
		return "truecolor"
	default:
		return mode
	}
}

// cycle returns the next/previous element after cur in choices (wrapping). If
// cur is absent, returns the first element.
func cycle(choices []string, cur string, dir int) string {
	for i, c := range choices {
		if c == cur {
			n := (i + dir + len(choices)) % len(choices)
			return choices[n]
		}
	}
	return choices[0]
}

func removeAt[T any](s []T, i int) []T {
	return append(s[:i:i], s[i+1:]...)
}

func insertAt[T any](s []T, i int, v T) []T {
	var zero T
	s = append(s, zero)
	copy(s[i+1:], s[i:])
	s[i] = v
	return s
}
