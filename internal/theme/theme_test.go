package theme

import (
	"os"
	"testing"
)

func TestHexToRGB(t *testing.T) {
	cases := []struct {
		in      string
		r, g, b int
		ok      bool
	}{
		{"#ffffff", 255, 255, 255, true},
		{"#000000", 0, 0, 0, true},
		{"7266EA", 0x72, 0x66, 0xEA, true}, // no leading '#'
		{"#7266ea", 0x72, 0x66, 0xEA, true},
		{"", 0, 0, 0, false},
		{"#fff", 0, 0, 0, false},     // shorthand not supported
		{"#gggggg", 0, 0, 0, false},  // non-hex
		{"#1234567", 0, 0, 0, false}, // too long
	}
	for _, c := range cases {
		r, g, b, ok := hexToRGB(c.in)
		if ok != c.ok || (ok && (r != c.r || g != c.g || b != c.b)) {
			t.Errorf("hexToRGB(%q) = (%d,%d,%d,%v), want (%d,%d,%d,%v)",
				c.in, r, g, b, ok, c.r, c.g, c.b, c.ok)
		}
	}
}

func TestColorCodeTrueColor(t *testing.T) {
	if got := colorCode("#ff0000", ModeTrueColor, false); got != "38;2;255;0;0" {
		t.Errorf("fg truecolor = %q, want 38;2;255;0;0", got)
	}
	if got := colorCode("#00ff00", ModeTrueColor, true); got != "48;2;0;255;0" {
		t.Errorf("bg truecolor = %q, want 48;2;0;255;0", got)
	}
	// An unparseable color yields no code at any mode.
	if got := colorCode("nope", ModeTrueColor, false); got != "" {
		t.Errorf("bad color = %q, want empty", got)
	}
	// ModeNone never emits a code.
	if got := colorCode("#ff0000", ModeNone, false); got != "" {
		t.Errorf("ModeNone = %q, want empty", got)
	}
}

func TestColorCodeDowngrade(t *testing.T) {
	// ansi256 fg uses the 38;5;N form; bg uses 48;5;N.
	if got := colorCode("#ffffff", ModeANSI256, false); got != "38;5;231" {
		t.Errorf("white ansi256 fg = %q, want 38;5;231", got)
	}
	if got := colorCode("#000000", ModeANSI256, true); got != "48;5;16" {
		t.Errorf("black ansi256 bg = %q, want 48;5;16", got)
	}
	// ansi (16-color) picks the nearest bright red (index 9) → base 90 + 1 = 91.
	if got := colorCode("#ff0000", ModeANSI, false); got != "91" {
		t.Errorf("red ansi fg = %q, want 91", got)
	}
	// bg adds 10 to the base: bright red bg = 101.
	if got := colorCode("#ff0000", ModeANSI, true); got != "101" {
		t.Errorf("red ansi bg = %q, want 101", got)
	}
}

func TestRGBToANSI16(t *testing.T) {
	cases := []struct {
		r, g, b int
		want    int
	}{
		{0, 0, 0, 0},        // black
		{255, 0, 0, 9},      // bright red
		{0, 255, 0, 10},     // bright green
		{255, 255, 255, 15}, // white
	}
	for _, c := range cases {
		if got := rgbToANSI16(c.r, c.g, c.b); got != c.want {
			t.Errorf("rgbToANSI16(%d,%d,%d) = %d, want %d", c.r, c.g, c.b, got, c.want)
		}
	}
}

func TestRGBToANSI256Grayscale(t *testing.T) {
	if got := rgbToANSI256(0, 0, 0); got != 16 {
		t.Errorf("black = %d, want 16", got)
	}
	if got := rgbToANSI256(255, 255, 255); got != 231 {
		t.Errorf("white = %d, want 231", got)
	}
	// A mid gray lands in the 232..255 grayscale ramp.
	if got := rgbToANSI256(128, 128, 128); got < 232 || got > 255 {
		t.Errorf("mid gray = %d, want in [232,255]", got)
	}
}

// clearNoColor removes NO_COLOR for the duration of a test (t.Setenv cannot
// unset, and setting it to "" still reads as present).
func clearNoColor(t *testing.T) {
	t.Helper()
	old, had := os.LookupEnv("NO_COLOR")
	if err := os.Unsetenv("NO_COLOR"); err != nil {
		t.Fatalf("unset NO_COLOR: %v", err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("NO_COLOR", old)
		}
	})
}

func TestResolveColorMode(t *testing.T) {
	t.Run("NO_COLOR forces none", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		if got := resolveColorMode("truecolor"); got != ModeNone {
			t.Errorf("NO_COLOR set = %v, want ModeNone", got)
		}
	})

	explicit := map[string]ColorMode{
		"none":      ModeNone,
		"ansi":      ModeANSI,
		"ansi256":   ModeANSI256,
		"truecolor": ModeTrueColor,
	}
	for pref, want := range explicit {
		t.Run("explicit "+pref, func(t *testing.T) {
			clearNoColor(t)
			if got := resolveColorMode(pref); got != want {
				t.Errorf("resolveColorMode(%q) = %v, want %v", pref, got, want)
			}
		})
	}

	t.Run("auto COLORTERM=truecolor", func(t *testing.T) {
		clearNoColor(t)
		t.Setenv("COLORTERM", "truecolor")
		if got := resolveColorMode("auto"); got != ModeTrueColor {
			t.Errorf("= %v, want ModeTrueColor", got)
		}
	})
	t.Run("auto TERM 256color", func(t *testing.T) {
		clearNoColor(t)
		t.Setenv("COLORTERM", "")
		t.Setenv("TERM", "xterm-256color")
		if got := resolveColorMode("auto"); got != ModeANSI256 {
			t.Errorf("= %v, want ModeANSI256", got)
		}
	})
	t.Run("auto dumb TERM none", func(t *testing.T) {
		clearNoColor(t)
		t.Setenv("COLORTERM", "")
		t.Setenv("TERM", "dumb")
		if got := resolveColorMode("auto"); got != ModeNone {
			t.Errorf("= %v, want ModeNone", got)
		}
	})
	t.Run("auto plain xterm ansi", func(t *testing.T) {
		clearNoColor(t)
		t.Setenv("COLORTERM", "")
		t.Setenv("TERM", "xterm")
		if got := resolveColorMode("auto"); got != ModeANSI {
			t.Errorf("= %v, want ModeANSI", got)
		}
	})
}

func TestResolveCharset(t *testing.T) {
	explicit := map[string]Charset{
		"ascii":    CharsetASCII,
		"nerdfont": CharsetNerdFont,
		"unicode":  CharsetUnicode,
	}
	for pref, want := range explicit {
		if got := resolveCharset(pref); got != want {
			t.Errorf("resolveCharset(%q) = %v, want %v", pref, got, want)
		}
	}

	t.Run("auto env hooks", func(t *testing.T) {
		t.Setenv("CC_STATUSLINE_ASCII", "1")
		if got := resolveCharset("auto"); got != CharsetASCII {
			t.Errorf("ASCII hook = %v, want CharsetASCII", got)
		}
	})
	t.Run("auto nerdfont hook", func(t *testing.T) {
		t.Setenv("CC_STATUSLINE_ASCII", "")
		t.Setenv("CC_STATUSLINE_NERDFONT", "1")
		if got := resolveCharset("auto"); got != CharsetNerdFont {
			t.Errorf("nerdfont hook = %v, want CharsetNerdFont", got)
		}
	})
	t.Run("auto default unicode", func(t *testing.T) {
		t.Setenv("CC_STATUSLINE_ASCII", "")
		t.Setenv("CC_STATUSLINE_NERDFONT", "")
		if got := resolveCharset("auto"); got != CharsetUnicode {
			t.Errorf("default = %v, want CharsetUnicode", got)
		}
	})
}

func TestLangIconsForCharset(t *testing.T) {
	// Nerd Font has language glyphs; the Go gopher is U+E626.
	nf := langIconsFor(CharsetNerdFont)
	if nf["go"] != "" {
		t.Errorf("go glyph = %q, want U+E626", nf["go"])
	}
	if nf["node"] == "" || nf["python"] == "" {
		t.Error("expected node and python glyphs in nerdfont charset")
	}
	// bun/deno have no standard glyph.
	if nf["bun"] != "" || nf["deno"] != "" {
		t.Error("bun/deno should have no glyph")
	}
	// Unicode and ASCII carry no language glyphs (avoid mojibake).
	if langIconsFor(CharsetUnicode) != nil {
		t.Error("unicode charset should have nil Lang map")
	}
	if langIconsFor(CharsetASCII) != nil {
		t.Error("ascii charset should have nil Lang map")
	}
}

func TestGlyphsLangIcon(t *testing.T) {
	th, err := Load("catppuccin-mocha", "nerdfont", "none", nil)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := th.Glyphs.LangIcon("go"); got != "" {
		t.Errorf("LangIcon(go) = %q, want U+E626", got)
	}
	if got := th.Glyphs.LangIcon("bun"); got != "" {
		t.Errorf("LangIcon(bun) = %q, want empty", got)
	}
	// A unicode theme resolves no language icon.
	thU, _ := Load("catppuccin-mocha", "unicode", "none", nil)
	if got := thU.Glyphs.LangIcon("go"); got != "" {
		t.Errorf("unicode LangIcon(go) = %q, want empty", got)
	}
}
