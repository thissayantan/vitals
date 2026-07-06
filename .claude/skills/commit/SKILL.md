---
name: commit
description: Create atomic commits using Conventional Commits + Gitmoji, tailored to the vitals (Go) project
allowed-tools: Bash(git add:*), Bash(git status:*), Bash(git diff:*), Bash(git commit:*), Bash(git reset:*), Bash(git log:*), Bash(git branch:*), Bash(git rev-parse:*), Bash(go build:*), Bash(go test:*), Bash(go vet:*), Bash(gofmt:*), Bash(golangci-lint:*)
---

# Atomic Commit Helper

## Current State

**Branch**: !`git branch --show-current`

**Status**:
!`git status --short`

**Staged changes**:
!`git diff --cached --stat`

**Unstaged changes**:
!`git diff --stat`

**Recent commits for style reference**:
!`git log --oneline -5`

## Your Task

Create small, atomic commits following the **Conventional Commits + Gitmoji** format.

### Commit Message Format

```
<gitmoji> <type>(<scope>): <description>
```

- Append `!` after the scope for a breaking change: `✨ feat(config)!: rename lines schema`,
  and add a `BREAKING CHANGE: <what changed>` footer.

### Gitmoji Reference

| Emoji | Type     | When to use                          |
| ----- | -------- | ------------------------------------ |
| ✨    | feat     | New feature / new segment            |
| 🐛    | fix      | Bug fix                              |
| 📝    | docs     | Documentation (README, DESIGN, docs) |
| ♻️    | refactor | Code restructuring, no behavior change |
| 🎨    | style    | Formatting / `gofmt` / lint-only changes |
| ⚡️    | perf     | Performance (matters on the render hot path) |
| ✅    | test     | Add/update tests, golden files       |
| 🔒️    | security | Security fix                         |
| 🔧    | chore    | Config/tooling (go.mod, .golangci.yml) |
| 👷    | ci       | CI / GitHub Actions / GoReleaser     |
| ⬆️    | deps     | Bump dependencies                    |
| 🔥    | remove   | Remove code/files                    |
| 🏗️    | arch     | Architectural change                 |
| 💄    | ui       | TUI / status-line visual changes     |
| 🔖    | release  | Version tag / release prep           |
| 🎉    | init     | Initial commit                       |

### Scope Options (vitals project)

Prefer the Go package or area touched:
`render`, `segments`, `theme`, `config`, `cost`, `git`, `runtime`, `cache`, `tui`, `claude`,
`cli`, `schema`, `install`, `release`, `ci`, `docs`, `deps`.

### Rules

1. **Atomic commits** — one logical change per commit.
2. **Group related files only** — different purposes → different commits. Examples of proper splitting:
   - New segment (`internal/segments/foo.go`) → 1 commit
   - Its schema entry + default-config wiring → can ride along (same logical change)
   - Theme addition → 1 commit
   - Docs update → 1 commit
   - A cross-cutting mechanical rename is one logical change → 1 commit is fine.
3. **Description under 72 chars**, concise.
4. **Imperative mood** — "add segment" not "added segment".
5. **Lowercase** after the colon; no trailing period.
6. **Checks before committing** (Go project):
   - `gofmt -l .` is clean (no unformatted files)
   - `go build ./...` succeeds
   - `go test ./...` passes (run when code, not just docs, changed)
   - `golangci-lint run` is clean if available
7. **No secrets** — never commit `.env`, tokens, or credentials.
8. **No build artifacts** — never commit the `vitals` binary, `dist/`, `vendor/`, or coverage output.
9. **Branch safety** — do not commit directly on `main`. If `git branch --show-current` is `main`,
   create/switch to a feature branch first (e.g. `feat/<short-name>`), unless the user said otherwise.

### Process

1. Analyze the staged/unstaged changes shown above.
2. If changes are mixed (e.g. a fix + a feature), split them into separate commits.
3. Run the relevant checks from Rule 6 for the change type.
4. For each logical group:
   - Stage only the related files with `git add <files>` (avoid `git add -A`).
   - Commit with the format below.
5. Use a HEREDOC for multi-line messages (body / `BREAKING CHANGE:` footer).
6. Always append the co-author trailer.

### Commit Command Format

```bash
git commit -m "$(cat <<'EOF'
<gitmoji> <type>(<scope>): <short description>

<optional body explaining what & why, wrapped ~72 cols>

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

### Examples

```
✨ feat(segments): add git segment with cached dirty state
🐛 fix(render): drop separator around hidden segments
⚡️ perf(cache): memoize runtime detection per session
♻️ refactor(theme): load palettes from embedded JSON
✅ test(segments): add golden tests for context bar
📝 docs(readme): add quick-start and config section
👷 ci(release): wire GoReleaser on tag push
⬆️ deps(tui): bump bubbletea to latest
🔧 chore(config): add golangci-lint settings
💄 ui(tui): live preview pane for the configurator
```

Now analyze the changes and create appropriate atomic commit(s).
