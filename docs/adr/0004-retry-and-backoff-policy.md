# ADR-0004: Retry and backoff policy

## Status

Accepted

## Context

Canvas API requests can fail transiently: 429 (rate limit), 403 with `X-Rate-Limit-Remaining: 0` (rate limit exhausted), and 5xx (server errors). The CLI needs automatic retry for these conditions to be reliable for automation, while not retrying errors that are permanent (404, 401, 422).

## Decision

Retry logic lives in `Client.doWithRetry`. The `ShouldRetry` function determines retryability:

| Status | Retry? | Condition |
|---|---|---|
| 429 | Always | Up to `maxRetries` |
| 403 | Only if `X-Rate-Limit-Remaining: 0` | Rate-limited 403, not permission 403 |
| 500–599 | Always | Transient server errors |
| Other | No | Permanent errors |

Delay calculation:
- Honor `Retry-After` header when present (Canvas sends this for rate limits).
- Otherwise use exponential backoff: base 1s, doubling per attempt, capped at 30s.
- Add jitter: random value between 0 and 25% of the delay, to avoid thundering herd.

Retries respect context cancellation. The retry count is configurable via `--retries` (default from config).

## Consequences

- Rate-limited requests back off automatically without user intervention.
- Permission errors (403 without rate limit signal) are not retried — they fail fast.
- Jitter prevents synchronized retry storms when multiple agents hit the same rate limit.
- Context cancellation aborts retry loops immediately.
