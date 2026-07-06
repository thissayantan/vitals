// Package theme handles colors, charsets, and per-segment styling.
//
// See DESIGN.md §5. A theme defines semantic roles (brand, ok/warn/crit, muted,
// accent, add, del, …) plus the context-bar gradient. Segments reference roles,
// never raw colors. Colors auto-downgrade truecolor → ansi256 → ansi → none and
// honor NO_COLOR. Charset auto-selects unicode/nerdfont/ascii glyph sets.
package theme

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	vitals "github.com/thissayantan/vitals"
)

// Color is a hex string ("#7266EA") or empty for "no color".
type Color string

// ColorMode is the resolved terminal color capability.
type ColorMode int

// Color-mode levels, from least to most capable.
const (
	ModeNone      ColorMode = iota // NO_COLOR / no styling
	ModeANSI                       // 16-color
	ModeANSI256                    // 256-color
	ModeTrueColor                  // 24-bit
)

// Charset selects the glyph family used for icons and bars.
type Charset int

// Glyph families selectable via the charset setting.
const (
	CharsetUnicode Charset = iota
	CharsetNerdFont
	CharsetASCII
)

// Style is a foreground/background color plus bold/dim attributes. The resolved
// color mode is carried unexported so Style.Render is self-contained (matching
// the CONTRIBUTING recipe: ctx.Theme.Style("accent").Render("x")).
type Style struct {
	Fg   Color `json:"fg,omitempty"`
	Bg   Color `json:"bg,omitempty"`
	Bold bool  `json:"bold,omitempty"`
	Dim  bool  `json:"dim,omitempty"`

	mode ColorMode
}

// Glyphs is the resolved icon/bar set for a charset.
type Glyphs struct {
	Brand     string // model marker ◆
	Branch    string // git branch ⎇
	BlockIcon string // 5h rate-limit ↻
	ResetIcon string // reset-time ↺
	Clock     string // duration ⏱
	Gear      string // worktree ⚙
	Flag      string // tasks ⚑

	BarFilled  string // context bar █
	BarEmpty   string // context bar ░
	TaskFilled string // task bar ▣
	TaskEmpty  string // task bar ▢

	// Lang maps a runtime language (as detected by internal/runtime) to its
	// icon. Only populated for the Nerd Font charset; nil for unicode/ascii,
	// where the runtime segment falls back to the language name alone.
	Lang map[string]string
}

// LangIcon returns the icon for a runtime language, or "" if this charset has
// none (unicode/ascii, or an unmapped language such as bun/deno).
func (g Glyphs) LangIcon(lang string) string { return g.Lang[lang] }

// Theme is a fully-resolved palette + charset + color mode.
type Theme struct {
	Name     string
	Roles    map[string]Style
	Gradient []Color
	Charset  Charset
	Mode     ColorMode
	Glyphs   Glyphs
}

// themeData is the on-disk JSON shape of a themes/*.json file.
type themeData struct {
	Name     string           `json:"name"`
	Roles    map[string]Style `json:"roles"`
	Gradient []Color          `json:"gradient"`
}

// Load resolves a theme by name with the given charset/colorMode preferences and
// optional per-role overrides (used when name == "custom"). All three string
// args accept "auto" to trigger environment detection.
func Load(name, charset, colorMode string, overrides map[string]Style) (*Theme, error) {
	mode := resolveColorMode(colorMode)
	cs := resolveCharset(charset)

	roles, gradient, err := loadPalette(name)
	if err != nil {
		return nil, err
	}
	for role, st := range overrides {
		roles[role] = st
	}
	// "none" theme and ModeNone both mean no styling.
	if name == "none" {
		mode = ModeNone
	}
	// Stamp the resolved mode onto every role so Style.Render is self-contained.
	for role, st := range roles {
		st.mode = mode
		roles[role] = st
	}
	glyphs := glyphsFor(cs)
	glyphs.Lang = langIconsFor(cs)
	return &Theme{
		Name:     name,
		Roles:    roles,
		Gradient: gradient,
		Charset:  cs,
		Mode:     mode,
		Glyphs:   glyphs,
	}, nil
}

// langIconsFor returns the per-language icon map for a charset. Only the Nerd
// Font charset has language glyphs; unicode/ascii return nil so the runtime
// segment falls back to the language name alone (no mojibake). Codepoints match
// those used by bundled Oh My Posh themes, so they render in the same Nerd Fonts
// users already have. bun/deno have no standard glyph and are omitted.
func langIconsFor(cs Charset) map[string]string {
	if cs != CharsetNerdFont {
		return nil
	}
	return map[string]string{
		"go":     "",
		"node":   "",
		"python": "",
		"rust":   "",
		"ruby":   "",
		"java":   "",
		"php":    "",
		"dotnet": "",
		"swift":  "",
		"elixir": "",
	}
}

// loadPalette reads a built-in theme's roles + gradient from the embedded FS.
// The "none"/"custom" pseudo-themes fall back to catppuccin-mocha's palette.
func loadPalette(name string) (map[string]Style, []Color, error) {
	file := name
	if name == "custom" || name == "none" || name == "" {
		file = "catppuccin-mocha"
	}
	data, err := vitals.Themes.ReadFile("themes/" + file + ".json")
	if err != nil {
		// Fall back to the default theme rather than failing the render.
		data, err = vitals.Themes.ReadFile("themes/catppuccin-mocha.json")
		if err != nil {
			return nil, nil, fmt.Errorf("theme %q not found and default missing: %w", name, err)
		}
	}
	var td themeData
	if err := json.Unmarshal(data, &td); err != nil {
		return nil, nil, fmt.Errorf("parse theme %q: %w", file, err)
	}
	if td.Roles == nil {
		td.Roles = map[string]Style{}
	}
	return td.Roles, td.Gradient, nil
}

// Style returns the style registered for role (with the theme's color mode
// stamped in). Unknown roles yield an empty, mode-aware style (plain text).
func (t *Theme) Style(role string) Style {
	if st, ok := t.Roles[role]; ok {
		return st
	}
	return Style{mode: t.Mode}
}

// NewStyle stamps the theme's color mode onto an ad-hoc style (e.g. a per-segment
// config override) so its Render produces correctly-downgraded escapes.
func (t *Theme) NewStyle(s Style) Style {
	s.mode = t.Mode
	return s
}

// Render wraps text in the ANSI escapes for this style, honoring the color mode.
// ModeNone (or an empty style) returns text unchanged.
func (s Style) Render(text string) string {
	if s.mode == ModeNone {
		return text
	}
	var codes []string
	if s.Bold {
		codes = append(codes, "1")
	}
	if s.Dim {
		codes = append(codes, "2")
	}
	if c := fgCode(s.Fg, s.mode); c != "" {
		codes = append(codes, c)
	}
	if c := bgCode(s.Bg, s.mode); c != "" {
		codes = append(codes, c)
	}
	if len(codes) == 0 {
		return text
	}
	return "\033[" + strings.Join(codes, ";") + "m" + text + "\033[0m"
}

// GradientColor returns the i-th gradient stop (clamped) as a Style with the
// theme's mode. Used by the shared bar renderer.
func (t *Theme) GradientColor(i int) Style {
	if len(t.Gradient) == 0 {
		return Style{mode: t.Mode}
	}
	if i < 0 {
		i = 0
	}
	if i >= len(t.Gradient) {
		i = len(t.Gradient) - 1
	}
	return Style{Fg: t.Gradient[i], mode: t.Mode}
}

// EmptyCell returns the dim style used for unfilled bar cells.
func (t *Theme) EmptyCell() Style {
	return Style{Fg: "#3c3c41", Dim: true, mode: t.Mode}
}

// ─── color code generation ──────────────────────────────────────────────

func fgCode(c Color, mode ColorMode) string { return colorCode(c, mode, false) }
func bgCode(c Color, mode ColorMode) string { return colorCode(c, mode, true) }

func colorCode(c Color, mode ColorMode, bg bool) string {
	r, g, b, ok := hexToRGB(string(c))
	if !ok {
		return ""
	}
	switch mode {
	case ModeTrueColor:
		if bg {
			return fmt.Sprintf("48;2;%d;%d;%d", r, g, b)
		}
		return fmt.Sprintf("38;2;%d;%d;%d", r, g, b)
	case ModeANSI256:
		n := rgbToANSI256(r, g, b)
		if bg {
			return fmt.Sprintf("48;5;%d", n)
		}
		return fmt.Sprintf("38;5;%d", n)
	case ModeANSI:
		n := rgbToANSI16(r, g, b) // 0..15
		base := 30
		if n >= 8 {
			base = 90 // bright
			n -= 8
		}
		if bg {
			base += 10
		}
		return strconv.Itoa(base + n)
	default:
		return ""
	}
}

// hexToRGB parses "#rrggbb" (or "rrggbb"). Returns ok=false for anything else.
func hexToRGB(s string) (r, g, b int, ok bool) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return 0, 0, 0, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	return int(v >> 16 & 0xff), int(v >> 8 & 0xff), int(v & 0xff), true
}

// rgbToANSI256 maps an RGB triple to the xterm 256-color cube/grayscale ramp.
func rgbToANSI256(r, g, b int) int {
	if r == g && g == b { // grayscale shortcut
		if r < 8 {
			return 16
		}
		if r > 248 {
			return 231
		}
		return 232 + (r-8)*24/247
	}
	q := func(v int) int { return v * 5 / 255 }
	return 16 + 36*q(r) + 6*q(g) + q(b)
}

// ansi16 is the standard 16-color palette in RGB for nearest-color matching.
var ansi16 = [16][3]int{
	{0, 0, 0}, {205, 0, 0}, {0, 205, 0}, {205, 205, 0},
	{0, 0, 238}, {205, 0, 205}, {0, 205, 205}, {229, 229, 229},
	{127, 127, 127}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0},
	{92, 92, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255},
}

// rgbToANSI16 returns the index (0..15) of the nearest 16-color entry.
func rgbToANSI16(r, g, b int) int {
	best, bestDist := 0, 1<<31-1
	for i, c := range ansi16 {
		dr, dg, db := r-c[0], g-c[1], b-c[2]
		d := dr*dr + dg*dg + db*db
		if d < bestDist {
			best, bestDist = i, d
		}
	}
	return best
}

// ─── environment detection ──────────────────────────────────────────────

func resolveColorMode(pref string) ColorMode {
	if _, noColor := os.LookupEnv("NO_COLOR"); noColor {
		return ModeNone
	}
	switch pref {
	case "none":
		return ModeNone
	case "ansi":
		return ModeANSI
	case "ansi256":
		return ModeANSI256
	case "truecolor":
		return ModeTrueColor
	}
	// auto
	switch strings.ToLower(os.Getenv("COLORTERM")) {
	case "truecolor", "24bit":
		return ModeTrueColor
	}
	term := os.Getenv("TERM")
	switch {
	case strings.Contains(term, "256color"):
		return ModeANSI256
	case term == "" || term == "dumb":
		return ModeNone
	default:
		return ModeANSI
	}
}

func resolveCharset(pref string) Charset {
	switch pref {
	case "ascii":
		return CharsetASCII
	case "nerdfont":
		return CharsetNerdFont
	case "unicode":
		return CharsetUnicode
	}
	// auto: preserve the prototype's env hooks.
	if os.Getenv("CC_STATUSLINE_ASCII") == "1" {
		return CharsetASCII
	}
	if os.Getenv("CC_STATUSLINE_NERDFONT") == "1" {
		return CharsetNerdFont
	}
	return CharsetUnicode
}

// glyphsFor returns the glyph set for a charset (mirrors the prototype's
// USE_ASCII / USE_NERDFONT / default symbol tables).
func glyphsFor(cs Charset) Glyphs {
	switch cs {
	case CharsetASCII:
		return Glyphs{
			Brand: "<>", Branch: "git:", BlockIcon: "", ResetIcon: "@",
			Clock: "", Gear: "", Flag: "",
			BarFilled: "#", BarEmpty: "-", TaskFilled: "#", TaskEmpty: "-",
		}
	case CharsetNerdFont:
		return Glyphs{
			Brand: "◆", Branch: " ", BlockIcon: "\U000f0519", ResetIcon: "\U000f051b",
			Clock: "\U000f0150 ", Gear: "\U000f013a ", Flag: "",
			BarFilled: "█", BarEmpty: "░", TaskFilled: "▣", TaskEmpty: "▢",
		}
	default: // unicode
		return Glyphs{
			Brand: "◆", Branch: "⎇ ", BlockIcon: "↻", ResetIcon: "↺",
			Clock: "⏱ ", Gear: "⚙ ", Flag: "⚑",
			BarFilled: "█", BarEmpty: "░", TaskFilled: "▣", TaskEmpty: "▢",
		}
	}
}
