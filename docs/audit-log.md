# Audit Log

Every remote mutation must append a local JSONL audit event.

Default path (OS-appropriate):

```text
Linux:   ~/.local/state/canvas-cli/audit.jsonl
macOS:   ~/Library/Application Support/canvas-cli/audit.jsonl
Windows: %LOCALAPPDATA%\canvas-cli\audit.jsonl
```

Config option:

```yaml
audit:
  enabled: true
  path: ~/.local/state/canvas-cli/audit.jsonl
```

## Event schema

```json
{
  "time": "2026-06-12T19:20:00Z",
  "schema_version": "2026-06-12",
  "command": "assignments.submit",
  "profile": "default",
  "base_url": "https://school.instructure.com",
  "method": "POST",
  "path": "/api/v1/courses/123/assignments/456/submissions",
  "resource": {
    "course_id": "123",
    "assignment_id": "456"
  },
  "request_hash": "sha256:...",
  "response_status": 200,
  "canvas_request_id": "...",
  "dry_run": false,
  "success": true
}
```

## Redaction

Never log:

- access token
- Authorization header
- full message bodies by default
- full assignment submission text by default
- raw uploaded file contents

Use hashes and metadata. Body logging is not included by default.
