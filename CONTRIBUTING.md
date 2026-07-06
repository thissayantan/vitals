# Contributing to vitals

Thanks for your interest! vitals is designed so the most common contribution — **adding a new
segment** — is easy and isolated.

## Development

```sh
go build ./...
go test ./...
golangci-lint run
# render against the sample session JSON:
go run ./cmd/vitals < testdata/session.sample.json
```

## Adding a segment (the canonical recipe)

A segment is one self-contained file. To add `foo`:

1. Create `internal/segments/foo.go`:
   ```go
   package segments

   func init() { Register(&fooSegment{}) }

   type fooSegment struct{}

   func (s *fooSegment) Type() string { return "foo" }

   func (s *fooSegment) Render(ctx *RenderCtx, cfg SegmentConfig) (string, bool) {
       // pull data from ctx.Session (and ctx.Git()/ctx.Runtime()/... if needed)
       // return ("", false) to hide the segment entirely
       return ctx.Theme.Style("accent").Render("foo"), true
   }
   ```
2. Add its options + `type` enum entry to `schema/vitals.schema.json`.
3. If it ships in the default layout, add it to the defaults in `internal/config`.
4. Add a golden test in `internal/segments/foo_golden_test.go` with a fixture and expected output.
5. Document it in the README segment table.

That's the whole surface area — no central switch statement to edit; the registry wires it up via
`init()`.

## Performance

The renderer runs on every Claude Code update. Keep segments cheap:
- No shelling out to `jq`. Use `encoding/json`.
- Expensive work (git, tool version lookups, transcript parsing) goes through a cached **provider**
  in `internal/{gitinfo,runtime,cost}` and `internal/cache`, computed lazily and once.
- Disabled segments must not trigger their provider.

## Commits & releases

- **Conventional Commits** (`feat:`, `fix:`, `docs:`, …) — drives the changelog.
- **SemVer** tags (`vX.Y.Z`) trigger GoReleaser via GitHub Actions.

## Code style

- `gofmt` / `golangci-lint` clean.
- Prefer data tables over branching (themes, runtime detection, pricing are all data).
