# Security

`canvas-cli` handles Canvas access tokens and can perform mutations such as assignment submission, discussion replies, inbox messages, grading, and course content updates.

## Token handling

- Tokens are never logged, printed to stdout, or included in error messages.
- Tokens are never passed via `--token` flag (to avoid shell history exposure).
- Tokens are read from stdin (`--token-stdin`) or environment variables (`--token-env`).
- Tokens are stored in the config file (`$XDG_CONFIG_HOME/canvas-cli/config.yaml`) with `0600` permissions.
- The `auth status` command reports only whether a token is present (`yes`/`no`), never the token value.
- Environment variable references are stored as `env:VARNAME` in the config, not the resolved value.

## Mutation safety

- `--dry-run` always allowed: previews the mutation without sending any request.
- `--read-only` flag or `CANVAS_READ_ONLY=1` env var blocks all write operations, even with `--confirm`.
- Low-risk writes (assignment submit, discussion reply, inbox send/archive) require `--confirm`.
- High-risk writes (grading, grade import, raw API writes) require `--confirm` and are logged to the audit log.

## Audit logging

Every mutation (when audit is enabled in config) writes a JSONL event to the audit log file at `$XDG_STATE_HOME/canvas-cli/audit.jsonl`. Each event includes:

- `time`: UTC timestamp (RFC 3339)
- `schema_version`: Output contract version
- `command`: The CLI command that triggered the mutation
- `profile`: Active config profile name
- `base_url`: Canvas instance URL (token excluded)
- `method`: HTTP method (POST, PUT, DELETE)
- `path`: API path (no query parameters or tokens)
- `resource`: Map of resource IDs (e.g., `course_id`, `assignment_id`)
- `request_hash`: SHA-256 hex digest of the request body, prefixed with `sha256:`
- `response_status`: HTTP status code from Canvas
- `canvas_request_id`: Canvas `X-Request-Id` header (if present)
- `dry_run`: Whether this was a dry-run
- `success`: Whether the operation succeeded

## Other rules

- Never store tokens in repository files.
- Support `env:CANVAS_TOKEN` config references for CI/CD environments.
- OAuth2 is planned for multi-user distribution scenarios.

Report security issues privately to the repository owner.
