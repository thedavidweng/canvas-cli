# Pagination and Rate Limits

Canvas list endpoints are paginated. `canvas-cli` should hide pagination for normal resource commands and expose it for debugging and raw API workflows.

## Pagination rules

Resource commands:

- auto-paginate by default
- default `--page-size 100`
- honor `--limit`
- support `--no-paginate`
- follow `Link` header URLs as opaque absolute URLs
- parse header names case-insensitively
- stop cleanly if the next link is absent

Raw API commands:

- default to a single response
- support `--paginate` to follow links
- support `--include-headers`
- support `--raw` for body-only output

## Rate limit rules

Canvas uses dynamic throttling. The client should capture:

- `X-Request-Cost`
- `X-Rate-Limit-Remaining`
- `Retry-After`

Retry policy:

- retry 429
- retry transient 5xx
- optionally retry 403 only when body/header clearly indicates rate-limit exhaustion
- respect `Retry-After`
- otherwise use bounded exponential backoff with jitter
- default max retries: 3
- default concurrency: 1

**Historical note:** Older Canvas versions and some Canvas instances return HTTP 403 (not 429) for rate limiting. The client should handle both: check for 429 first, then check 403 responses for the `X-Rate-Limit-Remaining: 0` header or a rate-limit error message in the body. Both should trigger the same retry behavior.

## Diagnostics

Human diagnostics go to stderr.

In JSON mode, include rate metadata where available:

```json
"rate_limit": {
  "request_cost": 1.2,
  "remaining": 998.8,
  "retry_after_seconds": null
}
```

## Concurrency

Default concurrency is one simultaneous request. Bulk downloads may support explicit concurrency:

```bash
canvas submissions download --course 123 --assignment 456 --out submissions --concurrency 2
```

Keep concurrency conservative. Do not parallelize normal list commands by default.
