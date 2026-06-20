# Contributing to canvas-cli

Thank you for your interest in contributing to canvas-cli.

## Getting started

```bash
git clone https://github.com/thedavidweng/canvas-cli.git
cd canvas-cli
mise install  # install tools pinned in mise.toml
go build ./cmd/canvas
go test ./...
```

Requires Go 1.26.4 or later.

## Development workflow

1. Fork the repository and create a feature branch.
2. Make your changes with tests.
3. Run `make check` (runs gofmt, go vet, go test, go build).
4. Open a pull request against `main`.

## Document precedence

When documentation conflicts, follow this order:

1. `docs/json-contract.md` -- JSON envelope shape, error codes, exit codes
2. `docs/command-spec.md` -- command syntax, flags, environment variables
3. `docs/architecture.md` -- package layout, type definitions, design rules
4. `docs/safety-model.md` -- safety levels, audit logging
5. Other docs, README, code comments

If a spec document and the code disagree, the spec document is authoritative. Fix the code, not the spec.

## Technical decisions

These are fixed and should not be revisited in pull requests:

- **Language**: Go 1.26+
- **CLI framework**: Cobra
- **Module path**: `github.com/thedavidweng/canvas-cli`
- **Binary name**: `canvas`
- **Config location**: OS config dir (`~/.config` on Linux, `~/Library/Application Support` on macOS, `%APPDATA%` on Windows)
- **State location**: OS state dir (`~/.local/state` on Linux, `~/Library/Application Support` on macOS, `%LOCALAPPDATA%` on Windows)
- **JSON output**: Stable envelope with `ok`, `data`, `error`, `meta` fields
- **String IDs**: All Canvas resource IDs are strings (Canvas returns string IDs to avoid JavaScript precision loss)

## Code style

- Standard Go formatting (`gofmt`).
- Use `goimports` with local prefix `github.com/thedavidweng/canvas-cli`.
- All exported functions must have doc comments.
- Error messages are lowercase, no trailing punctuation.
- Wrap errors with `fmt.Errorf("context: %w", err)`.
- Commands stay thin: parsing flags, calling shared packages, formatting output.
- Canvas API logic belongs in `internal/canvas/`, not in `internal/cli/`.

## Testing

Every change must include tests. Required categories:

- **Unit tests**: pure logic, no HTTP. Test helpers, parsers, formatters.
- **Integration tests**: use `testutil.MockCanvas` to simulate Canvas API responses. Cover success path, error path, pagination, rate limiting.
- **Golden file tests**: for human-readable output. Store expected output in `testdata/` and compare with `testutil` helpers.
- **Redaction tests**: verify tokens never appear in stdout, stderr, JSON output, or error messages.

Run before submitting:

```bash
go test ./... -race
go vet ./...
gofmt -l ./cmd ./internal   # must be empty
```

## JSON contract

Every command must produce a valid JSON envelope when `--json` is passed. See `docs/json-contract.md` for the canonical specification. Changes to the envelope shape require a `schema_version` bump.

## Safety model

Write commands must respect the safety model (`--dry-run`, `--confirm`, `--read-only`). See `docs/safety-model.md`. Never bypass safety checks.

## Commit messages

Use conventional commits:

- `feat: add new feature`
- `fix: resolve bug`
- `docs: update documentation`
- `refactor: restructure code`
- `test: add or update tests`
- `chore: maintenance tasks`

## Security

Never log or output API tokens. Report security issues privately to the repository owner. See `SECURITY.md` for full details.

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
