# Raw API Commands

Raw API commands provide direct access to any Canvas API endpoint. Use them for endpoints that do not have dedicated CLI commands, or for debugging and exploration.

## GET

```bash
canvas api get /api/v1/courses
canvas api get /api/v1/courses/123/assignments --query search_term=essay
canvas api get /api/v1/courses/123/assignments --paginate --json
canvas api get /api/v1/courses/123/assignments --include-headers --json
canvas api get /api/v1/courses/123/assignments --raw
```

Default behavior:

- one request only
- no auto-pagination unless `--paginate`
- JSON envelope unless `--raw`

## POST / PUT / DELETE

```bash
canvas api post /api/v1/courses/123/assignments --data @payload.json --dry-run
canvas api post /api/v1/courses/123/assignments --data @payload.json --confirm
canvas api put /api/v1/courses/123/assignments/456 --data '{"due_at":"2026-06-15T23:59:00Z"}' --confirm
canvas api delete /api/v1/courses/123/assignments/456 --confirm
```

Raw writes use the same safety model as dedicated commands:

- auth and timeout from resolved config
- retries with exponential backoff
- token redaction in all output
- `--dry-run` preview (method, path, query, payload summary)
- `--confirm` required
- audit log entry on success
