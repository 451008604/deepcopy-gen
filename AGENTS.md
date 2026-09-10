# Repository Guidelines

## Project Structure & Module Organization

This Go 1.21 module provides a deep-copy code generator. `main.go` implements the CLI; `scanner/` parses Go declarations into metadata defined in `types/`; `generator/` emits methods; `deepcopy/` supplies generic runtime helpers.

Unit tests live beside their implementations in `*_test.go`. `tests/runtime_test.go` verifies generated copy behavior. Main fixtures are consolidated in `testdata/types.go`; `testdata/external/` and `testdata/gamecore/` preserve cross-package cases, while `testdata/multifile/` exercises multiple source files. Generated methods reside in `deepcopy.dc.go` within each fixture package.

## Build, Test, and Development Commands

- `go build ./...` — compile the CLI and library packages.
- `go test ./...` — run unit, CLI, fixture compilation, and runtime tests.
- `go test ./... -cover` — inspect current coverage; no enforced percentage threshold is configured.
- `go test ./generator -run TestGenerate -count=1` — run focused generator tests without cached results.
- `go run . -dir ./testdata -dry-run` — preview fixture output.
- `go run . -dir ./testdata` — regenerate checked-in fixture methods.

Place flags before positional arguments. If the default Go cache is unavailable, prefix commands with `GOCACHE=/private/tmp/deepcopy-gen-gocache`.

## Coding Style & Naming Conventions

Keep code compatible with Go 1.21. Run `gofmt` on edited source files; use tabs and standard Go naming: exported identifiers use PascalCase, internal identifiers use camelCase, and package names are lowercase. Prefer direct implementations over unnecessary abstractions. Add concise comments explaining purpose or non-obvious behavior.

Change the generator or fixture source, then regenerate output; do not hand-edit `deepcopy.dc.go`. Inspect generated diffs before committing.

## Testing Guidelines

Use Go’s standard `testing` package with `TestFeature_Scenario` and `BenchmarkFeature` names. For generator fixes, cover emitted syntax and runtime isolation where relevant, including nil values, nested collections, cycles, and cross-package types. Keep specialized fixture packages intact. Run the full suite before submitting changes.

## Commit & Pull Request Guidelines

History uses Conventional Commit-style subjects such as `fix(generator): handle anonymous struct values` and `feat: support positional argument`. Keep commits focused and preserve unrelated working-tree changes.

PRs should describe the triggering case, resulting behavior, and validation commands. Link related issues when available and include regenerated fixtures when output changes.

## Agent Navigation

When `.codegraph/` exists, use `codegraph explore "<symbol or question>"` before text searches or file reads to locate or understand code. Do not initialize an index without user direction.
