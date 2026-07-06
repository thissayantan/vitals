# segments

One file per segment. Each registers itself in an `init()` via `Register`, so
adding a segment touches only one new file (plus the schema + config defaults).
See [../../DESIGN.md](../../DESIGN.md) §4 and [../../CONTRIBUTING.md](../../CONTRIBUTING.md)
for the extension recipe.

| File | Type | Renders | Hides when |
|---|---|---|---|
| `model.go` | `model` | `◆ Opus 4.8` (sparkle in Nerd Font) | never |
| `context.go` | `context` | `1M ████░░ 38% ctx` (brain glyph in Nerd Font) | badge < 200k |
| `block.go` | `block` | `↻ 5h 65% ↺8:10am` | five-hour limit absent / < 0 |
| `weekly.go` | `weekly` | `7d 12%` (calendar in Nerd Font) | seven-day limit absent / ≤ 0 |
| `cost.go` | `cost` | `$34.13` or estimated `≈ $X.XX` | value 0 |
| `duration.go` | `duration` | `⏱ 48m1s` | < 1000ms |
| `tasks.go` | `tasks` | `⚑ ▣▣▢░ 30%` (flag leads) | 0 tasks |
| `git.go` | `git` | `⎇ branch*` | not a repo |
| `diff.go` | `diff` | `± +2453/-439` (oct-diff in Nerd Font) | both 0 |
| `runtime.go` | `runtime` | `bun v1.3.14` (language logo in Nerd Font) | no language |
| `worktree.go` | `worktree` | `⚙ wt:name` | empty |
| `directory.go` | `directory` | `hirex` (folder in Nerd Font) | never |

`segments.go` holds the `Segment` interface, the registry, the shared `RenderCtx`,
and the small option/style helpers (`optString`, `optBool`, `optInt`, `styled`).
