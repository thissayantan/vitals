# vitals — Design & Specification

> **vitals** is a fast, modular status line for [Claude Code](https://code.claude.com).
> Tagline: *Your Claude Code session's vital signs, at a glance.*

This document is the **authoritative spec**. It is written for a fresh Claude Code session that
will implement the project from scratch. Read it top to bottom before writing code. It captures
*what to build*, *why*, and *the exact behavior to preserve* from the original Bash prototype.

---

## 1. What this is

Claude Code lets you replace the status bar at the bottom of the TUI with the output of any
command. Claude spawns that command on every status update and pipes a JSON object describing the
session to its **stdin**; whatever the command prints to **stdout** (1–N lines) becomes the status
line.

`vitals` is that command. It reads the session JSON, renders a styled, multi-line status bar made
of independent **segments** (modules), and exits. A second mode, `vitals config`, is an
in-terminal TUI for choosing, ordering, and styling those segments.

This is a clean-room rewrite of a working 511-line Bash prototype (`~/.claude/statusline.sh`). The
prototype proved the feature set; this project makes it **modular, configurable, fast, and
shippable** as a polished open-source tool.

### Goals
- **Preserve** every piece of information the prototype shows (see §4) — nothing the user relies on is lost.
- **Modular**: each segment is an isolated unit; users enable/disable/reorder/restyle them via config.
- **Extensible**: adding a new segment is one new file + one registry line — no edits elsewhere.
- **Fast**: the renderer runs on *every* update; cold start must be a few ms (see §3).
- **Friendly**: a TUI configurator with live preview, a safe `init` that wires up Claude Code, and a one-line installer.
- **Professional OSS**: MIT, semver, CI, GoReleaser, VHS demo, JSON-Schema-validated config.

### Non-goals
- Not a usage-analytics dashboard (that's `ccusage`'s job; we *render*, we don't aggregate history).
- Not a general shell prompt (that's starship/p10k); we target Claude Code's JSON contract specifically.

---

## 2. Platform contract (Claude Code statusLine)

From the official docs (https://code.claude.com/docs/en/statusline):

- Configured in `~/.claude/settings.json`:
  ```json
  { "statusLine": { "type": "command", "command": "vitals", "padding": 0 } }
  ```
- Claude pipes a **JSON object on stdin** each update. The command prints the status to **stdout**.
- Updates are **debounced at 300ms**; if a new update arrives while the previous command is still
  running, the in-flight process is **cancelled**. ⇒ keep render time well under 300ms.
- Optional `refreshInterval` (≥1000ms) re-renders on a timer for time-based segments (clock, reset countdown).
- First line's leading whitespace may be trimmed by Claude; `padding: 0` removes the default left pad.
- ANSI color/escape codes are supported. 256-color and truecolor work in modern terminals.

### stdin JSON shape (fields we consume)
The prototype reads these (all optional, with safe fallbacks):
```jsonc
{
  "model":          { "display_name": "Opus 4.8", "id": "claude-opus-4-8" },
  "context_window": { "used_percentage": 38, "context_window_size": 1000000 },
  "cost":           { "total_cost_usd": 34.13, "total_duration_ms": 2881000,
                      "total_lines_added": 2453, "total_lines_removed": 439 },
  "workspace":      { "current_dir": "/home/u/hirex", "project_dir": "/home/u/hirex" },
  "rate_limits":    { "five_hour":  { "used_percentage": 65, "resets_at": 1718000000 },
                      "seven_day":  { "used_percentage": 12, "resets_at": 1718500000 } },
  "worktree":       { "branch": "feat/hiring-pipeline", "name": "" },
  "transcript_path":"/home/u/.claude/projects/.../<session>.jsonl",
  "session_id":     "81bc496c-3174-4dbe-bb89-23128fe2d217",
  "cwd":            "/home/u/hirex"
}
```
> The exact schema may evolve; parse defensively (missing field ⇒ segment hides itself). A captured
> sample lives in `testdata/session.sample.json` (see §11) and drives golden tests.

---

## 3. Architecture

### One binary, several entry points
| Invocation | Purpose |
|---|---|
| `vitals` (stdin = session JSON) | **Render** the status line. The hot path. |
| `vitals config` | Launch the **TUI configurator** (bubbletea), with live preview. |
| `vitals init` | Safely merge the `statusLine` block into `~/.claude/settings.json` (backup first; never clobber existing keys). |
| `vitals doctor` | Diagnose: color support, fonts/charset, missing optional tools (git), config validity. |
| `vitals --version` / `-v` | Print version (injected at build time via ldflags). |
| `vitals print [--config f] [--input f]` | Dev helper: render from a file/sample for testing. |

Routing lives in `cmd/vitals/main.go`. No args + piped stdin ⇒ render. Keep the render path free of
any import that's only needed by the TUI (lazy/cheap startup).

### Package layout (`internal/`)
| Package | Responsibility |
|---|---|
| `claude` | stdin JSON types + defensive parsing. No business logic. |
| `config` | Config types, defaults, file discovery + merge, JSON-Schema-backed validation. |
| `segments` | The `Segment` interface, the **registry**, and one file per segment module. |
| `render` | Compose segments → lines; separators; the shared **bar** renderer; width/flex handling. |
| `theme` | Named themes, color model (truecolor/256/ansi auto-detect, `NO_COLOR`), charset (unicode/nerdfont/ascii), styling helpers. |
| `gitinfo` | Branch / dirty / ahead-behind via cheap `.git` reads + cached `git` exec. |
| `runtime` | Project language + version detection (lockfiles → tool/version), cached. |
| `cost` | Pricing table + transcript-based cost estimation for subscription users. |
| `cache` | Session-keyed on-disk cache (`$XDG_CACHE_HOME/vitals`) with TTLs. |
| `tui` | The bubbletea/huh/lipgloss configurator; reuses `render` + `theme` for an exact live preview. |

### The Segment interface (the heart of modularity)
```go
package segments

// RenderCtx is everything a segment may need, computed once per invocation and shared.
type RenderCtx struct {
    Session *claude.Session   // parsed stdin
    Theme   *theme.Theme      // resolved colors + charset
    Cache   *cache.Store      // session-keyed cache
    // lazily-populated providers (segments call these; results are memoized):
    Git     func() gitinfo.Info
    Runtime func() runtime.Info
    Cost    func() cost.Estimate
}

// SegmentConfig is the user's per-segment options (type-specific keys live in Options).
type SegmentConfig struct {
    Type    string
    Enabled bool
    Style   *theme.Style              // optional per-segment color override
    Options map[string]any            // e.g. {"showSha": true, "display": "bar"}
}

// Segment renders one piece of the status line.
type Segment interface {
    Type() string
    // Render returns the styled text and whether to show it.
    // show=false ⇒ the segment (and its surrounding separator) is omitted entirely.
    Render(ctx *RenderCtx, cfg SegmentConfig) (text string, show bool)
}

// Registry: each module registers itself in an init() so adding a module touches one file.
func Register(s Segment)         // called from each segment file's init()
func Get(typ string) (Segment, bool)
func All() []string
```
**Adding a new segment** = create `internal/segments/<name>.go`, implement `Segment`, call
`Register(&fooSegment{})` in `init()`, add its options to the JSON Schema and the config defaults.
Nothing else changes. Document this in `CONTRIBUTING.md` as the canonical extension recipe.

### Performance rules (non-negotiable; this runs every update)
1. Parse stdin with `encoding/json` natively. **Never shell out to `jq`.**
2. Minimize `git`: read `.git/HEAD` directly for the branch; only exec `git` for dirty/diff and
   **cache** the result (keyed by repo + index mtime). Set a short timeout on any exec.
3. Cache expensive providers (runtime detection, cost estimate, git dirty) under
   `$XDG_CACHE_HOME/vitals/<session_id>/…` with sane TTLs (see §6).
4. Compute each provider **lazily and once**: a segment that isn't enabled must not trigger its cost.
5. Render synchronously; no goroutine fan-out needed at this size, but providers must be cheap.
6. Target: `time vitals < testdata/session.sample.json` ≪ 50ms warm, ≪ 300ms cold.

---

## 4. Modules (mapped 1:1 from the prototype) + redesigned default order

Every module below exists in the prototype (`~/.claude/statusline.sh`). The "Source" column is the
data origin; "Hides when" is the visibility rule to preserve.

| # | type | Renders | Source | Hides when |
|---|------|---------|--------|------------|
| 1 | `model` | `◆ Opus 4.8` (brand-purple ◆ + bold name) | `model.display_name` (strip any `(… context)` suffix) | never |
| 2 | `context` | window badge `1M`/`200k` + gradient **bar** + `38% ctx` | `context_window.{context_window_size,used_percentage}` | badge hidden if size <200k |
| 3 | `block` | `↻ 5h 65% ↺8:10am` (5-hour rate-limit % + reset time) | `rate_limits.five_hour.{used_percentage,resets_at}` | `used_percentage < 0` |
| 4 | `weekly` | `7d 12%` | `rate_limits.seven_day.used_percentage` | value ≤ 0 |
| 5 | `cost` | `$34.13` (real) **or** teal `~$X.XX` (estimated) | `cost.total_cost_usd`; else estimate from transcript (§7) | value == 0 |
| 6 | `duration` | `⏱ 48m1s` / `5h 23m` | `cost.total_duration_ms` | < 1000ms |
| 7 | `tasks` | `▣▣▢░░ 30% ⚑` progress bar + flag | `~/.claude/tasks/<session_id>/*.json` (count + statuses) | 0 tasks |
| 8 | `git` | `⎇ feat/hiring-pipeline*` (branch, `*`=dirty; optional short SHA) | `worktree.branch` ⇒ `.git/HEAD` ⇒ `git rev-parse --short` | not a repo |
| 9 | `diff` | `+2453/-439` (green/red) | `cost.total_lines_{added,removed}` | both == 0 |
| 10 | `runtime` | `bun v1.3.14` (language + version) | lockfile detection → tool/version files (§8) | no language detected |
| 11 | `worktree` | `⚙ wt:name` | `worktree.name` | empty |
| 12 | `directory` | `hirex` (basename; option for full/truncated path) | `workspace.current_dir` (basename) | never |

### Redesigned default layout
The prototype's **line 2 order is backwards** (it starts with task progress and ends with the
directory). New default groups segments by meaning:

- **Line 1 — session & budget** *(identity → consumption → economics)*
  ```
  ◆ Opus 4.8 1M │ ████░░░░░░ 38% ctx │ ↻ 5h 65% ↺8:10am │ 7d 12% │ ⏱ 48m1s │ $34.13
    model              context           block(5h)         weekly   duration   cost
  ```
- **Line 2 — workspace & work** *(Place → Version control → Environment → Active work)*
  ```
  hirex │ ⎇ feat/hiring-pipeline* +2453/-439 │ bun v1.3.14 │ ▣▣▢░░ 30% ⚑
   dir       git              diff            runtime         tasks
  ```
  `worktree` sits between `directory` and `git` when present (it hides when empty).

This is only the **shipped default** — the whole point is that it lives in config and users reorder
freely. The default config (§5) encodes exactly this order.

---

## 5. Configuration

### Format & discovery
- **JSON** (Claude Code's native config language), validated by a published **JSON Schema**
  (`schema/vitals.schema.json`) for editor autocomplete + `vitals doctor` validation.
- Resolution precedence (first found wins, then deep-merged onto defaults):
  1. `./.vitals.json` (project-local)
  2. `$XDG_CONFIG_HOME/vitals/config.json` (default `~/.config/vitals/config.json`)
  3. built-in defaults (the redesigned layout above)
- `$VITALS_CONFIG` env var overrides discovery with an explicit path.

### Schema (shape)
```jsonc
{
  "$schema": "https://raw.githubusercontent.com/<owner>/vitals/main/schema/vitals.schema.json",
  "theme": "catppuccin-mocha",     // built-in name | "custom"
  "charset": "auto",               // auto | unicode | nerdfont | ascii
  "colorMode": "auto",             // auto | truecolor | ansi256 | ansi | none
  "separator": " │ ",              // between segments (theme may override)
  "lines": [
    { "segments": [
        { "type": "model" },
        { "type": "context", "options": { "display": "bar", "barWidth": 10 } },
        { "type": "block" },
        { "type": "weekly" },
        { "type": "duration" },
        { "type": "cost", "options": { "source": "auto" } }
    ] },
    { "segments": [
        { "type": "directory", "options": { "style": "basename" } },
        { "type": "worktree" },
        { "type": "git", "options": { "showSha": false } },
        { "type": "diff" },
        { "type": "runtime" },
        { "type": "tasks" }
    ] }
  ],
  "themeOverrides": { "model": { "fg": "#7266EA" } }  // used when theme == "custom"
}
```
Rules:
- **Order lives in `lines[].segments[]`.** Moving a segment = moving its entry. Multiple lines
  supported (1..N). Per-line `align`/`flex` is a later enhancement.
- Each segment entry: `type` (required), `enabled` (default `true` — lets users keep-but-disable),
  optional `style` override, and `options` (segment-specific; documented per module + in schema).
- `theme` selects a built-in palette; `theme: "custom"` + `themeOverrides` for per-segment colors.
- Color/charset auto-detect: truecolor via `COLORTERM`, honor `NO_COLOR`, downgrade gracefully;
  charset `auto` picks nerdfont glyphs only if a Nerd Font is likely (env hint), else unicode, with
  an explicit `ascii` escape hatch (preserves the prototype's `CC_STATUSLINE_ASCII/NERDFONT`).

### Built-in themes (initial set)
`catppuccin-mocha` (default, matches the user's terminal), `nord`, `tokyo-night`, `gruvbox`,
`rose-pine`, plus `none` (no color). Themes are data (embedded `themes/*.json` via `go:embed`).
A theme defines named roles (brand, ok/warn/crit, muted, accent, add, del) + the context-bar gradient.

---

## 6. Caching (`internal/cache`)
On-disk under `$XDG_CACHE_HOME/vitals/`. Keyed by `session_id` and/or a path hash. Preserve the
prototype's TTLs as defaults (make them config-tunable later):
| Provider | TTL | Key |
|---|---|---|
| git (branch+dirty) | 5s | repo path + `.git/index` mtime |
| runtime (lang+version) | 1h | project dir |
| cost estimate | 10s | transcript path hash |
| tasks | 3s | session_id |
Use atomic write (temp file + rename). No global lock needed if keys are per-session, but tolerate
concurrent renders (last-writer-wins is fine).

---

## 7. Cost (`internal/cost`)
Two paths (from the prototype):
- **API users**: `cost.total_cost_usd > 0` ⇒ render real `$X.XX`. Color tiers: ≥$10 crit, ≥$5 warn, else normal.
- **Subscription users**: field is 0 ⇒ estimate by parsing the transcript JSONL (`transcript_path`).
  Sum tokens per assistant message (`message.usage.{input_tokens,output_tokens,cache_read_input_tokens,
  cache_creation_input_tokens}`) × a **pricing table** keyed by model family. Render teal `~$X.XX`
  (the `~` signals an estimate). Cache 10s.

Pricing table is **data, not code** (embed `cost/pricing.json`; structure mirrors LiteLLM so it can
be refreshed). Seed values from the prototype (per 1M tokens, in/out/cache-read/cache-write):
- Opus 4.x: 5 / 25 / 0.50 / 6.25
- Sonnet 4.x: 3 / 15 / 0.30 / 3.75
- Haiku 4.x: 1 / 5 / 0.10 / 1.25
- (legacy Opus 3.x/4.1: 15/75; Haiku 2/3: 0.25/1.25)
Add a `source` option (`auto|cc|estimate`) and a note in README that estimates are approximate.

---

## 8. Runtime detection (`internal/runtime`)
Detect project language by lockfile/manifest, then resolve a version. Preserve the prototype's 12:
`bun.lock`→bun, `deno.json`→deno, `package.json`→node, `Cargo.toml`→rust, `go.mod`→go,
`pyproject.toml`/`requirements.txt`→python, `Gemfile`→ruby, `composer.json`→php, `*.csproj`→dotnet,
`pom.xml`→java, `mix.exs`→elixir, `Package.swift`→swift.
Version source order: pinned version files (`.nvmrc`, `.python-version`, `.ruby-version`, `.tool-versions`)
**before** execing the tool (`bun --version`, etc.). Cache 1h. Make the mapping a **declarative table**
(not 12 if-blocks) so new languages are one row.

---

## 9. TUI configurator (`vitals config`)
Built with **charmbracelet**: bubbletea (MVU loop), huh (forms/toggles), bubbles `list` (reorder),
lipgloss (styling). Features:
- List of segments per line; toggle on/off, reorder (move up/down), move between lines.
- Pick theme + charset + separator; per-segment style override.
- **Live preview** pane rendered with the *same* `render`+`theme` code the renderer uses, fed by
  `testdata/session.sample.json` (or the real last session if available) — so what you see is exact.
- Save writes `~/.config/vitals/config.json` (pretty-printed, with `$schema`). Offer to run `vitals init`.
Keep it keyboard-first and discoverable (help footer). This configurator is the project's signature
differentiator (ccstatusline's TUI is why it leads the category at 11k★).

---

## 10. Distribution

### `vitals init`
Implemented in Go (robust JSON merge, not shell). Reads `~/.claude/settings.json`, backs it up
(`settings.json.bak.<ts>`), deep-merges:
```json
{ "statusLine": { "type": "command", "command": "<abs path to vitals>", "padding": 0 } }
```
preserving all existing keys (hooks, etc.). Idempotent. Prints next steps. Never blind-overwrites.

### `install.sh` (one-liner)
Hardened POSIX script (model after zoxide/atuin). Steps:
1. `uname` → OS/arch → release asset name.
2. Resolve latest (or `--version`) from GitHub Releases API.
3. Download over HTTPS with `curl --proto '=https' --tlsv1.2 -fsSL` (wget fallback).
4. **Verify SHA256** against GoReleaser's `checksums.txt` (credibility win; most tools skip it).
5. Install to `~/.local/bin` (no sudo); `--bin-dir` override.
6. Print PATH check + `vitals init` + `vitals config` next steps. Whole body wrapped in a function
   called at the end (guards against truncated-download partial exec). Flags: `--version`,
   `--bin-dir`, `--no-modify-path`.
Usage in README: `curl -fsSL https://raw.githubusercontent.com/<owner>/vitals/main/install.sh | sh`.

### Release (`.goreleaser.yaml` + GitHub Actions)
GoReleaser on `git tag vX.Y.Z`: cross-compile linux/darwin/windows × amd64/arm64, archives +
`checksums.txt`, conventional-commit changelog, Homebrew tap formula, GitHub Release upload.
Inject version via ldflags. CI (`ci.yml`): `go test ./...`, `golangci-lint run`, and
`goreleaser release --snapshot --clean` dry-run on PRs.

---

## 11. Testing & fixtures
- `testdata/session.sample.json`: a realistic sample matching the screenshot (Opus 4.8, 1M ctx 38%,
  5h 65% reset 8:10am, 7d 12%, $34.13, branch `feat/hiring-pipeline`, +2453/-439, bun, dir `hirex`).
  **Capture a real one** early via the live setup if possible; otherwise use the provided sample.
- **Golden tests** per segment: fixture JSON in → expected ANSI string out (`*_golden.txt`). Run with
  `NO_COLOR` for stable plain-text goldens plus a colored variant.
- End-to-end: `echo "$(cat testdata/session.sample.json)" | vitals` matches the two-line golden.
- `vitals init` test: merge into a *copy* of a settings.json fixture; assert existing keys survive +
  backup created.
- Latency check in CI: assert render under a threshold.

---

## 12. Repo layout
```
vitals/
├── cmd/vitals/main.go            # entry + subcommand routing
├── internal/{claude,config,segments,render,theme,gitinfo,runtime,cost,cache,tui}/
├── schema/vitals.schema.json     # config JSON Schema
├── themes/*.json                 # embedded theme palettes
├── testdata/                     # sample session JSON + golden files
├── demo/*.tape                   # VHS scripts → README gif (regenerated in CI)
├── install.sh                    # one-line installer
├── .goreleaser.yaml
├── .github/workflows/{ci.yml,release.yml}
├── .golangci.yml .gitignore go.mod
├── README.md DESIGN.md CONTRIBUTING.md CHANGELOG.md LICENSE(MIT)
└── STARTING-PROMPT.md            # (handoff; can delete after bootstrapping)
```

## 13. Implementation order (suggested for the build session)
1. `claude` types + `testdata/session.sample.json` + parse.
2. `theme` (colors/charset) + `render` (bar + line compose) — get *something* printing.
3. `segments` registry + the 12 modules (start with model/context/directory/git, then the rest).
4. `config` load/merge/defaults + `schema/vitals.schema.json`; wire defaults to the redesigned layout.
5. Golden tests; `vitals print` dev helper; latency check.
6. `cache`, `gitinfo`, `runtime`, `cost` providers (with TTLs).
7. `vitals init` (+ test).
8. `tui` configurator with live preview.
9. `install.sh`, `.goreleaser.yaml`, CI, VHS demo, README with hero gif.
10. Polish: `vitals doctor`, more themes, docs.

Build incrementally; after each step, run `go build ./...`, `go test ./...`, and a manual
`echo … | go run ./cmd/vitals`.
