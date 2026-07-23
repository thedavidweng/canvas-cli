# ADR-0005: JSON envelope as public contract

## Status

Accepted

## Context

The CLI serves both humans (table output) and agents (machine-readable output). Agents need stable, parseable JSON to operate reliably. If the JSON shape changes between versions, agent scripts break.

## Decision

`--json` output follows a fixed envelope schema. The envelope is the public interface — its shape is a contract, not an implementation detail.

```json
{
  "ok": true,
  "data": <any>,
  "meta": {
    "schema_version": "2026-06-12",
    "command": "courses.list",
    "profile": "default",
    "base_url": "https://school.instructure.com",
    "pagination": { "next": "...", "previous": "..." },
    "rate_limit": { "remaining": 700, "request_cost": 1.0 }
  }
}
```

Error envelope:

```json
{
  "ok": false,
  "error": {
    "code": "CANVAS_AUTH_ERROR",
    "message": "...",
    "status": 401,
    "category": "auth",
    "retryable": false,
    "canvas_request_id": "..."
  },
  "meta": { "schema_version": "...", "command": "..." }
}
```

The `schema_version` field lets agents detect breaking changes. Error codes (`CANVAS_AUTH_ERROR`, `CANVAS_RATE_LIMIT`, etc.) are stable strings, not HTTP status numbers.

Exit codes are also part of the contract: 0 success, 3 config/auth, 6 network, 7 read-only block, 8 partial failure.

## Consequences

- Adding fields to `data` is safe; removing or renaming fields is a breaking change.
- Agents can rely on `ok`, `error.code`, and exit codes for control flow.
- Human output (tables) is secondary and can change freely.
- `--ndjson` streams the same envelope per line for paginated output.
