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

	"github.com/sayantan/vitals/internal/claude"
	"github.com/sayantan/vitals/internal/config"
	"github.com/sayantan/vitals/internal/render"
	"github.com/sayantan/vitals/internal/theme"
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
}

func newModel(cfg *config.Config) model {
	return model{
		cfg:     cfg,
		preview: previewSession(),
		status:  "↑/↓ move · space toggle · J/K reorder · t theme · c charset · [/] separator · s save · q quit",
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
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit

		case "up", "k":
			m.cursor--
			m.clampCursor()
		case "down", "j":
			m.cursor++
			m.clampCursor()

		case " ", "enter":
			m.toggleCurrent()
		case "J":
			m.moveCurrent(+1)
		case "K":
			m.moveCurrent(-1)

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
	}
	return m, nil
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
	b.WriteString("\n")
	b.WriteString(dim.Render(m.status))
	b.WriteString("\n")
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
			b.WriteString(fmt.Sprintf("  %s%s %s\n", pointer, box, name))
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
