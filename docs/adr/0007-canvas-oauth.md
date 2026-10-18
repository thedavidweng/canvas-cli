# 0007: Canvas OAuth authorization and refresh

Status: Accepted

## Context

Issue #16 requires usable Developer Key authorization without a personal access
token. PKCE helpers alone do not deliver that flow. Canvas public and confidential
clients have different secret requirements, and public-client refresh rotates
refresh tokens.

Protocol was verified against Instructure Canvas commit
`1c9f0bb8013ed69c4f2efe11fd483025469b7e6c`:

- [Controller grants and endpoints](https://github.com/instructure/canvas-lms/blob/1c9f0bb8013ed69c4f2efe11fd483025469b7e6c/app/controllers/oauth2_provider_controller.rb)
- [PKCE S256](https://github.com/instructure/canvas-lms/blob/1c9f0bb8013ed69c4f2efe11fd483025469b7e6c/lib/canvas/oauth/pkce.rb)
- [PKCE public clients](https://github.com/instructure/canvas-lms/blob/1c9f0bb8013ed69c4f2efe11fd483025469b7e6c/lib/canvas/oauth/grant_types/authorization_code_with_pkce.rb)
- [Developer Key secret requirements](https://github.com/instructure/canvas-lms/blob/1c9f0bb8013ed69c4f2efe11fd483025469b7e6c/lib/canvas/oauth/grant_types/base_type.rb)
- [Refresh and rotation](https://github.com/instructure/canvas-lms/blob/1c9f0bb8013ed69c4f2efe11fd483025469b7e6c/lib/canvas/oauth/grant_types/refresh_token.rb)
- [Token response](https://github.com/instructure/canvas-lms/blob/1c9f0bb8013ed69c4f2efe11fd483025469b7e6c/lib/canvas/oauth/token.rb)

## Decision

Use standard-library HTTP and existing PKCE helpers, with no new dependencies.
`auth oauth login` binds the explicitly registered fixed loopback address, emits
the browser URL to stderr, validates state, exchanges the code and atomically
persists tokens in the selected 0600 config. Client settings accept `env:NAME`.
No OS keychain or separate callback process is introduced.

Install a shared OAuth transport per command. Exact-origin requests refresh on
401 once and replay once if the body is replayable. Further requests share the
replacement bearer and refresh budget. Streaming requests are not replayed.
Token exchange and API redirects are not followed, preventing credential
forwarding and login-page confusion. Canvas requires HTTPS except on literal
loopback test origins. Token endpoint failures never expose response bodies.

Keep the old refresh token when the server omits a replacement; persist any
replacement before API replay. Manual refresh uses the same exchange/persistence.
Explicit PAT credentials win. Login with PAT/cookie and logout remove OAuth
tokens without discarding the configured Developer Key.

Add optional OAuth presence/expiry status fields to the existing JSON contract
and reuse `CANVAS_AUTH_ERROR`, preserving its authentication exit code.

## Consequences

Institutions must enable a Developer Key with the exact redirect URI; PKCE does
not bypass institutional authorization or make a confidential key public.
Mock-server command tests can verify login, reads, refresh and redaction without
institution credentials. Real consent and account reads remain a separate
acceptance item. Concurrent CLI processes do not coordinate refresh; avoid
simultaneously refreshing the same public-client profile because its server
rotates refresh tokens.
