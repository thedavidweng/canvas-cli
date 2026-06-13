# AGENTS.md

## Agent skills

### Issue tracker

GitHub Issues via `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Five canonical labels: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context layout. See `docs/agents/domain.md`.

### Verification

Before pushing code, always run:

```bash
go build ./...
go test ./...
golangci-lint run ./...
```

Lint config: `.golangci.yml` (golangci-lint v2). CI runs lint on every push to `main`.
