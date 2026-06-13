# Safety Model

`canvas-cli` is intended to perform real Canvas writes eventually. Safety must be designed before write commands are implemented.

## Safety levels

### Read operations

Read operations can run normally with token authentication.

Examples:

- list courses
- list modules
- list assignments
- get assignment details
- download files
- export course context

### Low-risk writes

Low-risk writes affect the authenticated user's own workflow or are easy to review.

Examples:

- send inbox message
- reply to discussion
- submit assignment
- add submission comment as self

Requirements:

- support `--dry-run`
- require explicit content input through flags/files/stdin
- emit audit log on execution
- reject empty body unless explicitly allowed

### High-risk writes

High-risk writes affect students, grades, course content, publication state, dates, or many records.

Examples:

- set grades
- batch grade import
- update due dates
- publish/unpublish content
- edit pages
- create announcements
- delete files
- update modules

Requirements:

- default to dry-run preview where feasible
- require `--confirm`
- require operation count confirmation for bulk operations
- support `--read-only` hard block
- emit audit log
- preserve Canvas response metadata
- show partial failures clearly

### Destructive operations

Destructive operations require `--confirm-delete`.

Examples:

- delete page
- delete discussion
- delete file
- delete module item

Requirements:

- require `--confirm-delete`
- require exact resource ID
- disallow wildcard deletes in v1/v2
- audit log mandatory

## Global safety flags

```text
--dry-run      build and display operation without sending mutation
--confirm      permit mutation
--read-only    block all mutation commands
```

Environment variable:

```text
CANVAS_READ_ONLY=1
```

`CANVAS_READ_ONLY=1` should override command flags and block writes.

## `--read-only` behavior

When `--read-only` is set (via flag or `CANVAS_READ_ONLY=1` env var):
1. Any write command exits immediately with exit code 7
2. The error message is: `"operation blocked by read-only mode"`
3. `--read-only` overrides `--confirm` — even with `--confirm`, writes are blocked
4. `--dry-run` is still allowed under `--read-only` (it doesn't perform writes)

## Interaction between `--dry-run`, `--confirm`, and `--read-only`

| Flags | Behavior |
|---|---|
| (none) | Prompt user interactively |
| `--dry-run` | Show preview, exit 0, no mutation |
| `--confirm` | Execute without prompt |
| `--dry-run --confirm` | Show preview, exit 0, no mutation |
| `--read-only` | Block with exit 7 |
| `--read-only --confirm` | Block with exit 7 |
| `--read-only --dry-run` | Show preview, exit 0 (allowed) |

## Audit log

Every remote mutation must append a local JSONL event.

Default path:

```text
Linux:   ~/.local/state/canvas-cli/audit.jsonl
macOS:   ~/Library/Application Support/canvas-cli/audit.jsonl
Windows: %LOCALAPPDATA%\canvas-cli\audit.jsonl
```

Audit event shape:

```json
{
  "time": "2026-06-12T19:20:00Z",
  "command": "grade.set",
  "profile": "default",
  "base_url": "https://school.instructure.com",
  "method": "PUT",
  "path": "/api/v1/courses/123/assignments/456/submissions/789",
  "resource": {
    "course_id": "123",
    "assignment_id": "456",
    "user_id": "789"
  },
  "request_hash": "sha256:...",
  "response_status": 200,
  "canvas_request_id": "...",
  "dry_run": false
}
```

Never log tokens or full sensitive message bodies by default. Log hashes and metadata. Provide `--audit-include-body` only if explicitly added later and documented.

## Dry-run output

Dry-run should show:

- intended method/path
- target resource IDs
- high-level diff or payload summary
- count of affected records
- whether the command would upload/download files

Dry-run must not send remote mutation requests.

## Bulk operations

Bulk write commands must generate a plan first.

Example:

```bash
canvas grade import --course 123 --assignment 456 --csv grades.csv --dry-run
```

Then:

```bash
canvas grade import --course 123 --assignment 456 --csv grades.csv --confirm
```

Partial failure handling:

- continue or stop behavior must be explicit
- default should stop on first write failure for high-risk operations unless `--continue-on-error` is set
- final JSON must include `ok: false` and category `partial_failure` when any item fails
