# JSON Contract

Stable JSON output is a public interface. Agents and scripts can depend on it across minor releases.

## Success envelope

```json
{
  "ok": true,
  "data": [],
  "meta": {
    "schema_version": "2026-06-12",
    "command": "courses.list",
    "profile": "default",
    "base_url": "https://school.instructure.com",
    "duration_ms": 123,
    "request_count": 2,
    "paginated": true,
    "page_size": 100,
    "limit": null,
    "rate_limit": {
      "request_cost": 1.23,
      "remaining": 987.65
    },
    "warnings": []
  }
}
```

## Error envelope

```json
{
  "ok": false,
  "error": {
    "code": "CANVAS_API_ERROR",
    "message": "Canvas API request failed",
    "category": "api",
    "retryable": false,
    "status": 400,
    "canvas_request_id": "abc123",
    "response_body": {
      "errors": []
    }
  },
  "meta": {
    "schema_version": "2026-06-12",
    "command": "assignments.list",
    "profile": "default",
    "base_url": "https://school.instructure.com",
    "duration_ms": 81,
    "request_count": 1
  }
}
```

## Entity normalization

Canvas objects can include many fields. Commands may return raw Canvas objects under `raw` when useful, but normalized top-level fields should be stable.

Example assignment object:

```json
{
  "id": "456",
  "course_id": "123",
  "name": "Essay 1",
  "description_html": "...",
  "due_at": "2026-06-30T23:59:00Z",
  "unlock_at": null,
  "lock_at": null,
  "published": true,
  "points_possible": 100,
  "submission_types": ["online_upload"],
  "has_submitted_submissions": false,
  "raw": {}
}
```

IDs are strings by default. The HTTP client requests Canvas string IDs.

### Why string IDs?

Canvas IDs can be large 64-bit integers. JavaScript's `Number.MAX_SAFE_INTEGER` is 2^53, so Canvas IDs above this threshold lose precision when parsed as JSON numbers by JavaScript-based tools (e.g., `jq`, browser DevTools, Node.js). To avoid this, the HTTP client must send the header `Accept: application/json+canvas-string-ids`, which causes Canvas to return all IDs as strings. This header is non-standard and Canvas-specific.

Not all Canvas instances support this header. If the response returns numeric IDs despite the header, the client should still function correctly but may log a warning in verbose mode.

## Exit codes

```text
0   success
1   generic runtime or API error
2   validation or usage error
3   auth or config error
4   permission denied
5   rate limit exhausted after retries
6   network or timeout error
7   safety policy blocked operation
8   partial failure in bulk operation
130 interrupted or cancelled
```

## Compatibility rules

- Adding new fields is allowed in minor releases.
- Removing or renaming fields requires a major version.
- Changing field type requires a major version.
- Error `code` strings are stable.
- Commands may include `raw` for Canvas-native fields.
- Human output may change more freely than JSON output.
