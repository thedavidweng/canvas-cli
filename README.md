<p align="center">
  <img src="public/icon.png" alt="canvas-cli" width="160" />
</p>

<h1 align="center">canvas-cli</h1>

<p align="center">
  Agent-friendly CLI for Canvas LMS.
</p>

<p align="center">
  <a href="https://github.com/thedavidweng/canvas-cli/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/thedavidweng/canvas-cli/ci.yml?branch=main&style=flat-square&label=ci" alt="CI"></a>
  <a href="https://github.com/thedavidweng/canvas-cli/releases"><img src="https://img.shields.io/github/v/release/thedavidweng/canvas-cli?style=flat-square" alt="Release"></a>
  <a href="https://github.com/thedavidweng/canvas-cli/blob/main/LICENSE"><img src="https://img.shields.io/github/license/thedavidweng/canvas-cli?style=flat-square" alt="License"></a>
  <img src="https://img.shields.io/badge/go-%3E%3D1.26-blue?style=flat-square" alt="Go">
</p>

`canvas-cli` gives students, teachers, scripts, and agents a stable terminal interface for Canvas LMS: predictable commands, JSON output, safe mutation gates, and a raw API escape hatch when the typed command surface is not enough.

## Why

Canvas automation often fails because browser flows, ad hoc exports, and one-off scripts are brittle. `canvas-cli` keeps the common LMS workflows small and scriptable while preserving enough structure for agents to reason about courses, assignments, submissions, files, and inbox state.

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

**Homebrew Cask (macOS/Linux):**

```shell
brew tap thedavidweng/tap
brew install --cask canvas
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
# Homebrew Cask
brew uninstall --cask canvas

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


## Infrastructure

- **CI/CD:** [cli-workflow-template](https://github.com/thedavidweng/cli-workflow-template) — reusable GitHub Actions workflows
- **Docs:** [site](https://github.com/thedavidweng/site) — landing page and documentation

## License

Apache License 2.0. See [LICENSE](LICENSE).
