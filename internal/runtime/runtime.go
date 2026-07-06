// Package runtime detects the project's language and version for the runtime
// segment.
//
// See DESIGN.md §8. Detection is a DECLARATIVE table (not 12 if-blocks) so
// adding a language is one row. Detect by lockfile/manifest, then resolve a
// version, preferring pinned version files before execing the tool. Cache 1h
// per project dir.
package runtime

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sayantan/vitals/internal/cache"
)

// Info is the detected language + version (either may be empty).
type Info struct {
	Language string
	Version  string
}

const execTimeout = 400 * time.Millisecond

// detector is one row of the detection table.
type detector struct {
	lang     string
	files    []string // any of these (exact name) marks the language
	globs    []string // …or any glob match in the project dir
	pinFiles []string // pinned-version files, read before execing
	pinTrim  string   // chars to strip from a pin file's contents
	cmd      []string // version command; first arg is the binary
	parse    func(string) string
}

// detectors is the ordered detection table. Order matters: bun before node
// (both have package.json-adjacent files), deno before node, etc.
var detectors = []detector{
	{
		lang:  "bun",
		files: []string{"bun.lock", "bun.lockb"},
		cmd:   []string{"bun", "--version"},
		parse: func(s string) string { return "v" + firstWord(s) },
	},
	{
		lang:  "deno",
		files: []string{"deno.json", "deno.jsonc"},
		cmd:   []string{"deno", "--version"},
		parse: func(s string) string { return "v" + field(firstLine(s), 1) },
	},
	{
		lang:     "node",
		files:    []string{"package.json"},
		pinFiles: []string{".nvmrc"},
		pinTrim:  "v \n\r",
		cmd:      []string{"node", "--version"},
		parse:    strings.TrimSpace, // node prints "vX.Y.Z"
	},
	{
		lang:  "rust",
		files: []string{"Cargo.toml"},
		cmd:   []string{"rustc", "--version"},
		parse: func(s string) string { return "v" + field(s, 1) },
	},
	{
		lang:  "go",
		files: []string{"go.mod"},
		cmd:   []string{"go", "version"},
		parse: func(s string) string { return strings.Replace(field(s, 2), "go", "v", 1) },
	},
	{
		lang:     "python",
		files:    []string{"pyproject.toml", "requirements.txt", "Pipfile", "setup.py", ".python-version"},
		pinFiles: []string{".python-version"},
		pinTrim:  " \n\r",
		cmd:      []string{"python3", "--version"},
		parse:    func(s string) string { return "v" + field(s, 1) },
	},
	{
		lang:     "ruby",
		files:    []string{"Gemfile", ".ruby-version"},
		pinFiles: []string{".ruby-version"},
		pinTrim:  " \n\r",
		cmd:      []string{"ruby", "--version"},
		parse:    func(s string) string { return "v" + field(s, 1) },
	},
	{
		lang:  "php",
		files: []string{"composer.json"},
		cmd:   []string{"php", "--version"},
		parse: func(s string) string { return "v" + field(firstLine(s), 1) },
	},
	{
		lang:  "dotnet",
		files: []string{"global.json"},
		globs: []string{"*.csproj", "*.sln"},
		cmd:   []string{"dotnet", "--version"},
		parse: func(s string) string { return "v" + strings.TrimSpace(s) },
	},
	{
		lang:  "java",
		files: []string{"pom.xml", "build.gradle", "build.gradle.kts"},
		cmd:   []string{"java", "-version"},
		parse: parseJavaVersion,
	},
	{
		lang:  "elixir",
		files: []string{"mix.exs"},
		cmd:   []string{"elixir", "--version"},
		parse: parseElixirVersion,
	},
	{
		lang:  "swift",
		files: []string{"Package.swift"},
		cmd:   []string{"swift", "--version"},
		parse: parseSwiftVersion,
	},
}

// Get detects language + version for the project dir, cached 1h.
func Get(dir string, c *cache.Store) Info {
	key := "runtime:" + dir
	out := c.Memo(key, cache.TTLRuntime, func() []byte {
		info := detect(dir)
		return []byte(info.Language + "\x00" + info.Version)
	})
	parts := strings.SplitN(string(out), "\x00", 2)
	info := Info{Language: parts[0]}
	if len(parts) > 1 {
		info.Version = parts[1]
	}
	return info
}

// detect runs the detection table against dir (uncached).
func detect(dir string) Info {
	for _, d := range detectors {
		if !d.matches(dir) {
			continue
		}
		return Info{Language: d.lang, Version: d.resolveVersion(dir)}
	}
	return Info{}
}

func (d detector) matches(dir string) bool {
	for _, f := range d.files {
		if fileExists(filepath.Join(dir, f)) {
			return true
		}
	}
	for _, g := range d.globs {
		if matches, _ := filepath.Glob(filepath.Join(dir, g)); len(matches) > 0 {
			return true
		}
	}
	return false
}

// resolveVersion prefers a pinned-version file, then execs the version command.
func (d detector) resolveVersion(dir string) string {
	for _, pf := range d.pinFiles {
		if data, err := os.ReadFile(filepath.Join(dir, pf)); err == nil {
			v := strings.Trim(string(data), d.pinTrim)
			if v != "" {
				return "v" + v
			}
		}
	}
	if len(d.cmd) == 0 {
		return ""
	}
	out, ok := runVersion(d.cmd)
	if !ok || strings.TrimSpace(out) == "" {
		return ""
	}
	if d.parse != nil {
		return d.parse(out)
	}
	return strings.TrimSpace(out)
}

// runVersion execs a version command with a short timeout, merging stderr (some
// tools, e.g. java, print version to stderr).
func runVersion(argv []string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), execTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", false
	}
	return string(out), true
}

// ─── small parse helpers ──────────────────────────────────────────────────

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func firstWord(s string) string { return field(s, 0) }

// field returns the n-th whitespace-separated field of s (0-indexed), or "".
func field(s string, n int) string {
	fs := strings.Fields(s)
	if n < 0 || n >= len(fs) {
		return ""
	}
	return fs[n]
}

// parseJavaVersion extracts the quoted version from `java -version` output,
// e.g. `openjdk version "21.0.2" …` → "v21.0.2".
func parseJavaVersion(s string) string {
	line := firstLine(s)
	if a := strings.IndexByte(line, '"'); a >= 0 {
		if b := strings.IndexByte(line[a+1:], '"'); b >= 0 {
			return "v" + line[a+1:a+1+b]
		}
	}
	return ""
}

// parseElixirVersion finds the "Elixir X.Y.Z" line.
func parseElixirVersion(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Elixir") {
			return "v" + field(line, 1)
		}
	}
	return ""
}

// parseSwiftVersion finds "Swift version X.Y.Z".
func parseSwiftVersion(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, "Swift version") {
			return "v" + field(line, 3)
		}
	}
	return ""
}
