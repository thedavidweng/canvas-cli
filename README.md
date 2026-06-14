# canvas-cli

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![CI](https://github.com/thedavidweng/canvas-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/thedavidweng/canvas-cli/actions/workflows/ci.yml)

Agent-friendly CLI for Canvas LMS. Stable JSON output, predictable exit codes, safe mutation gates, raw API escape hatch.

## Quickstart

### Install

Run the following on macOS or Linux:

```shell
curl -fsSL https://raw.githubusercontent.com/thedavidweng/canvas-cli/main/install.sh | sh
```

Run the following on Windows:

```shell
powershell -ExecutionPolicy ByPass -c "irm https://raw.githubusercontent.com/thedavidweng/canvas-cli/main/install.ps1 | iex"
```

The installer detects Homebrew automatically and uses it when available (recommended for easy upgrades). Otherwise it downloads the binary to `~/.local/bin`.

<details>
<summary>Other installation methods</summary>

**Homebrew (macOS/Linux):**

```shell
brew tap thedavidweng/tap
brew install canvas
```

**Go:**

```shell
go install github.com/thedavidweng/canvas-cli/cmd/canvas@latest
```

**Manual download:** grab the archive for your platform from the [latest GitHub Release](https://github.com/thedavidweng/canvas-cli/releases/latest), extract it, and place the `canvas` binary on your `PATH`.

</details>

### Set up

```shell
canvas auth login
```

The CLI walks you through selecting your Canvas instance and authenticating. You can use an access token (recommended) or a session cookie (experimental, for schools that disable tokens).

Then try it:

```shell
canvas courses list
canvas assignments list --course 123 --json
```

### Uninstall

```shell
# Homebrew
brew uninstall canvas

# install.sh
curl -fsSL https://raw.githubusercontent.com/thedavidweng/canvas-cli/main/install.sh | sh -s uninstall

# Go
rm "$(go env GOPATH)/bin/canvas"
```

Remove config if desired: `rm -rf ~/.config/canvas-cli`

## Documentation

- [Authentication & Configuration](docs/auth.md) — token, cookie, OAuth, profiles, env vars
- [Command Reference](docs/command-spec.md) — full command inventory with examples
- [JSON Contract](docs/json-contract.md) — envelope schema, exit codes, `--json` output
- [Safety Model](docs/safety-model.md) — read-only mode, dry-run, confirm gates
- [Architecture](docs/architecture.md) — codebase structure and design decisions
- [Contributing](CONTRIBUTING.md) — development setup and guidelines

## License

Apache License 2.0. See [LICENSE](LICENSE).
