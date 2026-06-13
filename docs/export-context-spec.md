# `canvas courses export-context` Specification

This is the most complex read command. It aggregates data from multiple Canvas API endpoints into a single JSON file for offline review and agent consumption.

## Usage

```bash
canvas courses export-context --course COURSE_ID --out context.json
canvas courses export-context --course COURSE_ID --out context.json --include modules,assignments,files
canvas courses export-context --course COURSE_ID --out context.json --since 2026-01-01 --json
```

## Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--course` | string | required | Course ID |
| `--out` | string | stdout | Output file path |
| `--include` | string | all | Comma-separated list of sections to include |
| `--since` | string | (none) | Only include items updated after this date (ISO 8601) |
| `--download-files` | string | (none) | Directory to download file attachments into |
| `--json` | bool | false | Wrap in standard JSON envelope |

## Sections and API calls

The command fetches sections in this order. Each section is isolated — a failure in one section does not abort others.

| Section | API endpoint | Notes |
|---|---|---|
| `course` | `GET /api/v1/courses/{id}?include[]=term&include[]=total_scores` | Always fetched first |
| `tabs` | `GET /api/v1/courses/{id}/tabs` | Helps discover enabled tools |
| `modules` | `GET /api/v1/courses/{id}/modules?include[]=items&include[]=content_details&per_page=100` | Auto-paginate. If inline items omitted by Canvas, fallback to `GET /api/v1/courses/{id}/modules/{module_id}/items` |
| `assignments` | `GET /api/v1/courses/{id}/assignments?per_page=100&order_by=due_at` | Auto-paginate |
| `assignment_groups` | `GET /api/v1/courses/{id}/assignment_groups?per_page=100` | |
| `files` | `GET /api/v1/courses/{id}/files?per_page=100` | Metadata only unless `--download-files` |
| `pages` | `GET /api/v1/courses/{id}/pages?per_page=100` then `GET /api/v1/courses/{id}/pages/{url}` for each | Get body content |
| `announcements` | `GET /api/v1/announcements?context_codes[]=course_{id}` | Auto-paginate. (Fallback: `discussion_topics?only_announcements=true` if caller lacks global view permission) |
| `discussions` | `GET /api/v1/courses/{id}/discussion_topics?only_announcements=false&per_page=100` | Exclude announcements |
| `submissions` | `GET /api/v1/courses/{id}/students/submissions?student_ids[]=all&per_page=100` or per-assignment | Only if user has permission |
| `grades` | `GET /api/v1/courses/{id}/enrollments?user_id=self&include[]=total_scores` | Student's own grades |

## Output JSON shape

```json
{
  "course": { ... },
  "tabs": [ ... ],
  "modules": [ ... ],
  "assignments": [ ... ],
  "assignment_groups": [ ... ],
  "files": [ ... ],
  "pages": [ ... ],
  "announcements": [ ... ],
  "discussions": [ ... ],
  "submissions": [ ... ],
  "grades": [ ... ],
  "_export_meta": {
    "generated_at": "2026-06-12T19:20:00Z",
    "course_id": "123",
    "sections_requested": ["modules", "assignments", ...],
    "sections_succeeded": ["modules", "assignments", ...],
    "sections_failed": [],
    "warnings": [],
    "request_count": 15,
    "duration_ms": 3200
  }
}
```

## Error isolation

- If a section fails with 403 (permission denied), record a warning and continue.
- If a section fails with 401, abort the entire export (auth is broken).
- If a section fails with a network error, retry per the standard retry policy, then record a warning and continue.
- The final output includes `sections_failed` so the consumer knows what's missing.
- Exit code: 0 if all requested sections succeeded, 8 (partial failure) if some sections failed.

## `--json` envelope

When `--json` is used, the export data goes into `data` and the standard envelope wraps it:

```json
{
  "ok": true,
  "data": { "course": {}, "modules": [], ... },
  "meta": { "command": "courses.export-context", ... }
}
```

When `--json` is NOT used, the raw export JSON (without envelope) is written to `--out` or stdout.

## Concurrency and Pagination

- **Concurrency**: The command defaults to a concurrency of 1. It may be configurable to a very small value, but Canvas's official rate-limiting guidelines recommend single-concurrent clients for safety.
- **Pagination**: The implementation MUST strictly follow Canvas official pagination rules. Always inspect the `Link` header, and treat returned link URLs as opaque absolute URLs (do not attempt to parse or construct the next page URL manually).
