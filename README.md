<div align="center">

# vitals

**Your Claude Code session's vital signs, at a glance.**

A fast, modular status line for [Claude Code](https://code.claude.com) — model, context, rate
limits, cost, git, and more — rendered as composable segments you fully control.

<!-- Demo gif: generate with `vhs demo/demo.tape` (produces docs/demo.gif), then uncomment: -->
<!-- ![vitals demo](docs/demo.gif) -->

[![CI](https://github.com/sayantan/vitals/actions/workflows/ci.yml/badge.svg)](https://github.com/sayantan/vitals/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/sayantan/vitals)](https://github.com/sayantan/vitals/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

</div>

---

## Why vitals

- ⚡ **Fast** — a single static Go binary, ~5ms cold start. It renders on *every* Claude Code update
  (debounced 300ms), so speed matters. No Node/Python runtime to install.
- 🧩 **Modular** — every piece of info is a **segment**. Enable, disable, reorder, and restyle them.
- 🎨 **Themed** — built-in themes (catppuccin, nord, tokyo-night, …), truecolor/256/ansi auto-detect,
  Nerd Font or ASCII fallback, `NO_COLOR` respected.
- 🛠️ **Configurable, visually** — `vitals config` is an in-terminal TUI with a live preview.
- 📦 **One-line install** — `curl … | sh`, then `vitals init` wires it into Claude Code for you.

## Quick start

```sh
# Install (no runtime required). Add --nerdfont if your terminal uses a Nerd Font
# to enable language/status icons (e.g. the Go gopher on the runtime segment).
curl -fsSL https://raw.githubusercontent.com/sayantan/vitals/main/install.sh | sh

# Wire it into Claude Code (safely merges ~/.claude/settings.json, backup first)
vitals init

# Customize, visually
vitals config
```

That's it — open Claude Code and your new status line is live.

<details>
<summary>Manual setup (no installer)</summary>

Add to `~/.claude/settings.json`:
```json
{ "statusLine": { "type": "command", "command": "vitals", "padding": 0 } }
```
</details>

## What it shows

Two lines by default (each fully reorderable):

```
◆ Opus 4.8 │ 1M ████░░░░░░ 38% ctx │ ↻ 5h 65% ↺8:10am │ 7d 12% │ ⏱ 48m1s │ $34.13
hirex │ ⎇ feat/hiring-pipeline* │ +2453/-439 │ bun v1.3.14 │ ▣▣▢░░░░░░░ 30% ⚑
```

Each segment **smart-hides** when its data is zero, empty, or unavailable (no branch outside a
repo, no cost on a fresh session, etc.), so the line stays tidy.

| Segment | Shows |
|---|---|
| `model` | active model |
| `context` | context-window size, usage bar, % used |
| `block` | 5-hour rate-limit usage + reset time |
| `weekly` | 7-day rate-limit usage |
| `cost` | session cost (real for API, estimated for subscription) |
| `duration` | session wall-clock time |
| `directory` | current project directory |
| `worktree` | git worktree name (when present) |
| `git` | branch + dirty state |
| `diff` | lines added / removed |
| `runtime` | project language + version |
| `tasks` | task-list progress |

See **[DESIGN.md](DESIGN.md)** for the full spec and **[config docs](#configuration)** for options.

## Configuration

Config is JSON, validated by a [JSON Schema](schema/vitals.schema.json). Discovery order:
`./.vitals.json` → `~/.config/vitals/config.json` → built-in defaults.

```jsonc
{
  "$schema": "https://raw.githubusercontent.com/sayantan/vitals/main/schema/vitals.schema.json",
  "theme": "catppuccin-mocha",
  "charset": "auto",
  "separator": " │ ",
  "lines": [
    { "segments": [ {"type":"model"}, {"type":"context"}, {"type":"block"},
                    {"type":"weekly"}, {"type":"duration"}, {"type":"cost"} ] },
    { "segments": [ {"type":"directory"}, {"type":"worktree"}, {"type":"git"},
                    {"type":"diff"}, {"type":"runtime"}, {"type":"tasks"} ] }
  ]
}
```

- **Order** = the order of entries in `lines[].segments[]`.
- **Disable** a segment: set `"enabled": false` (or remove it).
- **Restyle**: add `"style": {...}`, or set `"theme": "custom"` + `themeOverrides`.

Or just run `vitals config` and do it visually.

### Themes & charset

Built-in themes: `catppuccin-mocha` (default), `nord`, `tokyo-night`, `gruvbox`, `rose-pine`, and
`none` (plain). `charset` is `auto` | `unicode` | `nerdfont` | `ascii`, and `colorMode` is `auto` |
`truecolor` | `ansi256` | `ansi` | `none`. Colors auto-downgrade to whatever the terminal supports,
and `NO_COLOR` is always honored. The `nerdfont` charset also prefixes the runtime segment with a
per-language icon (Go, Node, Python, Rust, …); `unicode`/`ascii` show the language name alone.

### Environment overrides

These env vars (from the original prototype) still work, overriding `auto` detection:

| Variable | Effect |
|---|---|
| `NO_COLOR` | disable all color |
| `CC_STATUSLINE_ASCII=1` | force the ASCII glyph set |
| `CC_STATUSLINE_NERDFONT=1` | force Nerd Font glyphs |
| `CC_STATUSLINE_24H=1` | 24-hour reset clock instead of am/pm |
| `VITALS_CONFIG=<path>` | use an explicit config file |

Run **`vitals doctor`** to see exactly what was detected (color mode, charset, theme, config source).

## Adding a segment

Segments are isolated and registry-based — a new one is **one file**. See
[CONTRIBUTING.md](CONTRIBUTING.md).

## Acknowledgements

Inspired by the Claude Code status-line ecosystem — [ccstatusline](https://github.com/sirmalloc/ccstatusline),
[claude-powerline](https://github.com/Owloops/claude-powerline), [ccusage](https://github.com/ryoppippi/ccusage) —
and by [starship](https://starship.rs) / [oh-my-posh](https://ohmyposh.dev) for config design.

## License

[MIT](LICENSE)
