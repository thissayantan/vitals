# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/) and the project adheres to
[Semantic Versioning](https://semver.org/). Releases are generated from
[Conventional Commits](https://www.conventionalcommits.org/) via GoReleaser.

## [Unreleased]

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
