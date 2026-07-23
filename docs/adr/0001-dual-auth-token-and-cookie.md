# ADR-0001: Dual auth model (token + cookie)

## Status

Accepted

## Context

Canvas LMS supports two authentication mechanisms:

1. **API token** — a long-lived bearer token generated from Canvas account settings. Simple, stateless, works everywhere.
2. **Session cookie** — the browser session cookie (`canvas_session`) plus a CSRF token (`csrf_token`). Required for some institutions that disable API token generation, and convenient for users who already have a browser session.

The CLI needs to support both so it works across all Canvas deployments.

## Decision

Support both token and cookie auth on the same `Client`. When both are configured, **token takes precedence** over cookie.

Token auth sends `Authorization: Bearer <token>`. Cookie auth sends `Cookie: <cookie>` and `X-CSRF-Token` on unsafe methods (POST, PUT, PATCH, DELETE). The CSRF token is cached from response headers when not explicitly provided.

Cross-origin redirects strip all auth headers (`Cookie`, `Authorization`, `X-CSRF-Token`) to prevent credential leakage to third-party hosts. Unsafe methods do not follow redirects at all.

`DoURL` (used for pagination links and file download URLs) only sends auth headers when the target host matches the configured base URL host.

## Consequences

- Token auth is preferred for automation and agents (no CSRF complexity, no session expiry).
- Cookie auth requires CSRF token management and session expiry detection.
- The redirect policy is per-request when cookie auth is active, to avoid mutating the shared `http.Client`.
- Session expiry is detected via redirect analysis and surfaced as `CANVAS_SESSION_EXPIRED` with a re-auth message.
