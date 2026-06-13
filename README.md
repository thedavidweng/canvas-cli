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

### 1. Get your Canvas access token

1. Open your Canvas instance in a browser (e.g. `https://school.instructure.com`)
2. Click **Account** -> **Settings**
3. Under **Approved Integrations**, click **+ New Access Token**
4. Give it a name (e.g. `canvas-cli`) and click **Generate Token**
5. Copy the token — you won't see it again

### 2. Log in

```bash
canvas auth login
```

The CLI will walk you through it:

```
Profile name (default): school
Canvas Instance URL (e.g. https://school.instructure.com): https://school.instructure.com
Generate an access token at: https://school.instructure.com/profile/settings
  Account -> Settings -> Approved Integrations -> New Access Token

Access Token: ********
Verifying credentials...

Authenticated as: Jane Doe (jane@school.edu)

Credentials saved to profile "school" in ~/.config/canvas-cli/config.yaml
```

For scripting, use flags instead:

```bash
# From stdin (safe — no shell history leak)
echo "YOUR_TOKEN" | canvas auth login --base-url https://school.instructure.com --token-stdin

# From environment variable
export CANVAS_TOKEN="YOUR_TOKEN"
canvas auth login --base-url https://school.instructure.com --token-env CANVAS_TOKEN
```

### Multiple accounts

Use `--profile` to manage multiple institutions or users:

```bash
# Set up two schools
canvas auth login --profile school1 --base-url https://school1.instructure.com
canvas auth login --profile school2 --base-url https://school2.instructure.com

# Switch default profile
canvas auth use school1

# Or use a specific profile inline
canvas --profile school2 courses list

# Or via environment variable
CANVAS_PROFILE=school2 canvas courses list

# List all profiles
canvas auth profiles
```

### 3. Explore your courses

```bash
canvas courses list
canvas modules list --course 123
canvas assignments list --course 123
canvas files list --course 123
```

### 4. Export course context for agents

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
