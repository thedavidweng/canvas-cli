# JSON Envelope Schema

Canonical specification: [`docs/json-contract.md`](docs/json-contract.md).

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
    "status": 400
  },
  "meta": {
    "schema_version": "2026-06-12",
    "command": "assignments.list"
  }
}
```

## Error codes

| Code | Category | HTTP Status | Description |
|---|---|---|---|
| `CANVAS_API_ERROR` | `api` | 500, 502, 503, other | Generic Canvas API error |
| `CANVAS_AUTH_ERROR` | `auth` | 401 | Authentication failed or token expired |
| `CANVAS_PERMISSION_DENIED` | `permission` | 403 | Insufficient permissions for the requested operation |
| `CANVAS_NOT_FOUND` | `not_found` | 404 | Requested resource does not exist |
| `CANVAS_VALIDATION_ERROR` | `validation` | 422 | Request body failed Canvas validation |
| `CANVAS_NETWORK_ERROR` | `network` | N/A | Network-level failure (connection refused, DNS, timeout) |
| `PARTIAL_FAILURE` | `partial_failure` | N/A | Some operations in a bulk request failed |

The `error` object may also contain:

- `canvas_request_id`: The `X-Request-Id` header from Canvas, useful for support tickets.
- `response_body`: The raw JSON body from the Canvas API error response.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Generic runtime or API error |
| 2 | Validation or usage error |
| 3 | Auth or config error |
| 4 | Permission denied |
| 5 | Rate limit exhausted after retries |
| 6 | Network or timeout error |
| 7 | Safety policy blocked operation |
| 8 | Partial failure in bulk operation |
| 130 | Interrupted or cancelled |

Compatibility: adding fields is non-breaking. Removing or renaming fields requires a major version.
