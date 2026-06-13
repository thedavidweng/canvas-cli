# canvas-cli

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![CI](https://github.com/thedavidweng/canvas-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/thedavidweng/canvas-cli/actions/workflows/ci.yml)

Agent-friendly CLI for Canvas LMS. Stable JSON output, predictable exit codes, safe mutation gates, raw API escape hatch.

The repository is `canvas-cli`. The installed binary is `canvas`.

## Install

**Homebrew (macOS/Linux):**

```bash
brew tap thedavidweng/tap
brew install canvas
```

**Go:**

```bash
go install github.com/thedavidweng/canvas-cli/cmd/canvas@latest
```

## Getting Started

### 1. Authenticate

```bash
# Pass token via stdin (recommended — avoids shell history leaks)
echo "YOUR_TOKEN" | canvas auth login --base-url https://school.instructure.com --token-stdin

# Or via environment variable
export CANVAS_TOKEN="YOUR_TOKEN"
canvas auth login --base-url https://school.instructure.com --token-env CANVAS_TOKEN
```

### 2. Explore your courses

```bash
canvas courses list
canvas modules list --course 123
canvas assignments list --course 123
canvas files list --course 123
```

### 3. Export course context for agents

```bash
canvas courses export-context --course 123 --out course-context.json
```

## Key Features

### JSON output for automation

All commands support `--json` for stable, machine-readable output:

```bash
canvas assignments list --course 123 --bucket unsubmitted --json
```

Output uses a [stable envelope schema](docs/json-contract.md):

```json
{
  "ok": true,
  "data": [],
  "meta": {
    "schema_version": "2026-06-12",
    "command": "assignments.list"
  }
}
```

### Safe mutations

Write operations support `--dry-run` and `--confirm` gates:

```bash
canvas assignments submit --course 123 456 --file paper.pdf --dry-run
canvas assignments submit --course 123 456 --file paper.pdf --confirm
```

### Raw API passthrough

Call any Canvas API endpoint directly:

```bash
canvas api get /api/v1/courses
canvas api get /api/v1/courses/123/assignments --paginate --json
```

## Documentation

| Document | Description |
|---|---|
| [COMMANDS.md](COMMANDS.md) | Full command inventory with examples |
| [JSON_SCHEMA.md](JSON_SCHEMA.md) | JSON envelope contract and exit codes |
| [SECURITY.md](SECURITY.md) | Security rules and audit logging |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Contributor guidelines |
| [CHANGELOG.md](CHANGELOG.md) | Version history |
| [docs/](docs/) | Architecture, auth, safety model, workflows, and more |

## License

Apache License 2.0. See [LICENSE](LICENSE).
