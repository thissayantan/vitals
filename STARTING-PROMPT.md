# Starting prompt for the implementation session

Open a **fresh Claude Code session in this directory**
(`/home/sayantan/everything/code/personal/vitals`) and paste the prompt below.

This repo is a scaffold: the full spec is in `DESIGN.md`, the structure and stubs are in place, and
the binary already builds + runs (skeleton). Your job is to implement it.

---

```
Build out `vitals`, a modular status line for Claude Code. This repo is already scaffolded — read
DESIGN.md first; it is the authoritative spec (architecture, the 12 segments to implement with
their exact data sources and "hides when" rules, the redesigned default two-line layout, the config
schema, performance rules, TUI, installer, and release pipeline). Also read CONTRIBUTING.md for the
"add a segment" recipe and cmd/vitals/main.go for the existing routing skeleton.

Context: this is a clean-room rewrite of a working 511-line Bash prototype at
~/.claude/statusline.sh. Preserve every piece of information that prototype shows; do NOT lose any
segment behavior. The user runs Ghostty + zsh on Linux, Catppuccin Mocha theme (the default theme
file is themes/catppuccin-mocha.json). If you want to inspect the original behavior, read
~/.claude/statusline.sh — but implement fresh in Go per DESIGN.md, do not port the bash.

Constraints:
- Language: Go (single static binary). The render path (no args, JSON on stdin) must stay fast —
  stdlib-only where possible, no shelling out to jq, cache expensive providers (DESIGN.md §3, §6).
- Keep `go build ./...`, `go test ./...`, and `go run ./cmd/vitals < testdata/session.sample.json`
  green at every step. Go is NOT installed on this machine yet — install Go ≥1.24 first.
- Don't touch the user's live ~/.claude/statusline.sh or ~/.claude/settings.json during development.
  `vitals init` must operate on settings.json safely (backup + JSON merge, never clobber), and you
  should test it against a COPY, not the real file, until the user opts in.

Suggested build order (DESIGN.md §13):
1. internal/claude — types + defensive Parse; validate against testdata/session.sample.json.
2. internal/theme + internal/render — get the two lines printing (bar + separators), reusing the
   embedded themes/*.json.
3. internal/segments — the registry + all 12 modules (start with model, context, directory, git;
   then block, weekly, cost, duration, diff, runtime, worktree, tasks). Each is one file with an
   init() Register(), honoring its visibility rule.
4. internal/config — Defaults() encoding the redesigned layout, discovery + deep-merge + JSON-Schema
   validation against schema/vitals.schema.json.
5. Golden tests per segment (NO_COLOR plain-text goldens) + an end-to-end two-line golden.
6. internal/cache, gitinfo, runtime, cost providers with the documented TTLs.
7. `vitals init` (+ test on a settings.json fixture copy).
8. internal/tui — the `vitals config` configurator with a LIVE PREVIEW that reuses internal/render.
9. Finalize install.sh, .goreleaser.yaml, the CI workflows, and a VHS demo (demo/demo.tape →
   docs/demo.gif). Fill in README badges and the demo gif.
10. Polish: `vitals doctor`, add the remaining themes (nord, tokyo-night, gruvbox, rose-pine, none).

Before opening a PR / publishing: run `go test ./...`, `golangci-lint run`, and
`goreleaser release --snapshot --clean`; confirm `time vitals < testdata/session.sample.json` is
well under the 300ms render budget. Don't create the GitHub repo or run `git init` until I ask.

Work incrementally, keep the build green, and check in with me after step 5 (a rendering MVP) before
going deep on the TUI and release pipeline.
```

---

## Notes for the implementer
- **Module path** in `go.mod` is `github.com/sayantan/vitals` (placeholder owner). Update it and the
  URLs in README / schema `$id` / install.sh `REPO` / `.goreleaser.yaml` once the real GitHub owner
  is known.
- **Repo vs binary name:** binary is `vitals`; repo can be `vitals` (with the tagline + GitHub topics
  `claude-code` / `statusline` / `tui` for discoverability), since the name itself isn't keyword-SEO.
- The captured `testdata/session.sample.json` mirrors the user's real status line (Opus 4.8, 1M ctx
  38%, 5h 65% reset 8:10am, 7d 12%, $34.13, branch feat/hiring-pipeline, +2453/-439, bun, dir hirex).
  Capturing a *real* sample from a live session is even better for goldens.
