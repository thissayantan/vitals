# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/) and the project adheres to
[Semantic Versioning](https://semver.org/). Releases are generated from
[Conventional Commits](https://www.conventionalcommits.org/) via GoReleaser.

## [Unreleased]

## [0.2.1] - 2026-07-07

### Added
- Installer can set up a Nerd Font: when none is detected it offers to install one
  (default Yes, interactive), or via `--nerdfont` / `--install-font` / `--font <name>`.
  Skips when a Nerd Font is already present; prints the required "select it in your
  terminal" step (which no installer can automate).

### Fixed
- `block` segment: add a space between the reset icon and the reset time
  (`↺8:10am` → `↺ 8:10am`).

## [0.2.0] - 2026-07-07

### Added
- Redesigned Nerd Font icon set: sparkle model marker, a leading brain glyph on
  `context`, a calendar on `weekly`, an oct-diff glyph on `diff`, a folder on
  `directory`, and an estimate marker (`≈`) on `cost`. The `tasks` flag now
  leads its segment (icon-first, consistent with every other segment).
- `cost` gains a `mode` option (`auto` | `subscription` | `api`) — subscription
  keeps the platform-reported number but marks it estimated (legacy `source`
  still accepted).
- `vitals config` TUI: `o` edits a segment's options live, `a`/`x` add/remove a
  segment, `p` cycles preset layouts.
- Layout presets (`full` | `minimal` | `compact`) and
  `vitals init --preset <name>` / `--seed-config` to write a starter config.

### Changed
- Default line 2 orders `runtime` before `diff`.
- Charset stays opt-in: Unicode is the portable default; `--nerdfont` (installer)
  or `charset: nerdfont` enables the richer glyphs.

## [0.1.0] - 2026-07-07

### Added
- Renderer with all 12 segments (model, context, block, weekly, cost, duration, tasks, git, diff,
  runtime, worktree, directory), each honoring its smart-hide rule.
- Theme engine: 6 built-in themes (catppuccin-mocha, nord, tokyo-night, gruvbox, rose-pine, none)
  with truecolor → ansi256 → ansi → none auto-downgrade, `NO_COLOR` support, and unicode/nerdfont/
  ascii charsets.
- Cached providers: git (`.git/HEAD` + 5s dirty cache), runtime detection (declarative 12-language
  table, 1h cache), and transcript-based cost estimation (embedded pricing table, 10s cache).
- Runtime segment language icons: in the Nerd Font charset the segment prefixes a per-language glyph
  (Go, Node, Python, Rust, Ruby, Java, PHP, .NET, Swift, Elixir). Unicode/ASCII charsets are
  unchanged (name only).
- JSON config with discovery + deep-merge + JSON-Schema validation; redesigned two-line default
  layout.
- `vitals config` — bubbletea TUI configurator with a live preview rendered by the real renderer.
- `vitals init` — safe `~/.claude/settings.json` wiring (timestamped backup + atomic JSON merge).
- `vitals doctor` — environment/color/charset/dependency/config diagnostics.
- `vitals print` dev helper; per-segment golden tests, end-to-end golden, and a render-latency guard.
- One-line `install.sh` (checksum-verified) and GoReleaser cross-compilation pipeline.
