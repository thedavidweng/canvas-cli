# ADR-0003: Audit logging for mutations

## Status

Accepted

## Context

When the CLI performs Canvas writes, especially grading and content edits, there must be a local record of what was done, when, and whether it succeeded. This is critical for teaching teams that need to trace grade changes, and for agents where the human operator needs to verify what the agent did.

## Decision

Every mutation appends a JSONL event to a local audit log at the OS state directory (`canvas-cli/audit.jsonl`). The auditor is append-only with file locking (`sync.Mutex`).

Event fields: `time`, `schema_version`, `command`, `profile`, `base_url`, `method`, `path`, `resource` (IDs), `request_hash`, `response_status`, `canvas_request_id`, `dry_run`, `success`.

**Never logged**: access tokens, Authorization headers, full request bodies, full message text, file contents. Request bodies are hashed with SHA-256 and stored as `sha256:<hex>`. This proves the request content without storing sensitive data.

Audit is opt-in via config (`audit.enabled: true`). When disabled, `WriteEvent` is a no-op.

## Consequences

- Audit events prove what was sent without leaking sensitive content.
- The hash allows correlation with Canvas's own request IDs (`canvas_request_id`) for cross-referencing.
- The JSONL format is grep-friendly and append-safe.
- File permissions are `0600`; parent directory is `0700`.
