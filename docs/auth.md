# Authentication and Configuration

## Configuration precedence

1. explicit flags
2. environment variables
3. config file
4. defaults

## Environment variables

```text
CANVAS_BASE_URL=https://school.instructure.com
CANVAS_TOKEN=...
CANVAS_COOKIE=...
CANVAS_PROFILE=default
CANVAS_CONFIG=/path/to/config.yaml
```

`CANVAS_BASE_URL` is the Canvas root URL. Accept accidental `/api/v1` suffixes and normalize internally.

## Config file

Default path (OS-appropriate):

```text
Linux:   ~/.config/canvas-cli/config.yaml
macOS:   ~/Library/Application Support/canvas-cli/config.yaml
Windows: %APPDATA%\canvas-cli\config.yaml
```

Example:

```yaml
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    token: env:CANVAS_TOKEN
    timeout: 30s
    retries: 3
    page_size: 100
```

Support `env:VAR_NAME` token references to avoid storing tokens directly.

## Token auth

Local automation can use Canvas access tokens. Document the Canvas UI path for personal/local usage:

```text
Account -> Settings -> Approved Integrations -> New Access Token
```

The README must also state that broad multi-user applications should use OAuth2. Manual token entry from other users is not appropriate for a distributed multi-user app.

## Session cookie auth (experimental)

**Warning**: Session cookie auth is experimental and fallback-only. Use token auth or OAuth2 when possible.

For a task-oriented walkthrough, see [Log in with a session cookie](how-to/log-in-with-a-session-cookie.md).

Session cookie auth is intended for students whose schools disable access token generation. It reads your browser's Canvas login cookie, which grants full account access and bypasses SSO/2FA boundaries.

### Limitations

- Cookies expire when your browser session ends or after idle timeout (typically 8-24 hours).
- Write commands (POST/PUT/DELETE) require a CSRF token. Without one, only read commands work.
- Cookie auth is less secure than token auth — anyone with the cookie can act as you.

### Interactive login

```bash
canvas auth login
# Select "Session cookie (experimental)" when prompted
```

The CLI will attempt to extract cookies from your browser automatically. If extraction fails, you can:
- Try another browser
- Enter the cookie manually (copy from DevTools -> Application -> Cookies)

Override browser auto-detection with `--browser`:

```bash
canvas auth login --browser firefox
```

### Scripted/CI login

```bash
# Read cookie from stdin
echo "$COOKIE" | canvas auth login --base-url URL --cookie-stdin

# Read cookie from environment variable
canvas auth login --base-url URL --cookie-env MY_COOKIE_VAR

# Read cookie from file (must be 0600 permissions)
canvas auth login --base-url URL --cookie-file /path/to/cookie

# CSRF token (optional, enables write commands)
canvas auth login --base-url URL --cookie-stdin --csrf-token-stdin
canvas auth login --base-url URL --cookie-env MY_COOKIE --csrf-token-env MY_CSRF
```

### Config file

```yaml
profiles:
  default:
    base_url: https://canvas.school.edu
    # cookie: env:CANVAS_COOKIE   # experimental, fallback only
    # csrf_token: env:CANVAS_CSRF # extracted alongside cookie
```

### Session expiry

When your cookie session expires, commands will return:

```
session expired. Re-authenticate: canvas auth login
```

Re-run `canvas auth login` to get a fresh cookie.

### Security

- Cookie values are stored in the config file with the same permissions model as tokens (0600).
- `env:VAR_NAME` references are supported to avoid storing cookies in plaintext.
- Cookie values are never displayed in `auth status`, `--json`, or doctor output.
- Cross-origin redirects strip all auth headers (Cookie, Authorization, X-CSRF-Token).

## OAuth2

OAuth2 uses a Canvas Developer Key with authorization code and S256 PKCE.
The institution must enable the key and register the exact redirect URI,
including the fixed port. Secretless authorization requires a key configured
as a **public client**; confidential keys require a client secret too.

Create a 0600 config file with a profile such as:

```yaml
current_profile: school
profiles:
  school:
    base_url: https://school.instructure.com
    oauth:
      client_id: env:CANVAS_CLIENT_ID
      redirect_uri: http://127.0.0.1:8765/oauth/callback
      # Confidential keys only:
      # client_secret: env:CANVAS_CLIENT_SECRET
```

Run `canvas auth oauth login` and open the URL printed to stderr in your browser.
The command binds the registered loopback address before displaying the URL,
checks callback state, exchanges the code with its PKCE verifier, and saves
access/refresh tokens and expiry. `--login-timeout` defaults to `5m`; Ctrl-C
cancels the command. The redirect must use HTTP with literal `127.0.0.1` or
`[::1]`, a fixed nonzero port and a callback path. Canvas itself must use HTTPS;
HTTP Canvas origins are accepted only for literal loopback test servers.

Client ID, client secret, access token and refresh token accept `env:NAME`
references. Login and refresh save newly issued tokens into the existing config
using an atomic replacement with mode 0600. Client environment references and
other profiles/settings are preserved. Refresh-token rotation is persisted;
if Canvas omits a replacement refresh token, the previous one is retained.

API commands refresh on a 401 at most once per invocation and replay the request
once when the body can be recreated. Pagination and clients within the invocation
share that refresh budget and use the replacement access token. Streaming bodies
cannot be replayed; run `canvas auth refresh` before retrying an upload.
OAuth requests require the exact configured Canvas origin (scheme, hostname and
port); redirects are returned without following them. Use a separate profile
for another origin. A PAT from flags/environment/profile takes precedence over
OAuth. PAT/cookie login and logout clear stored OAuth tokens.

`canvas auth refresh` refreshes explicitly. `canvas auth status --json` reports
`auth_method: "oauth"`, token presence, refresh-token presence and `expires_at`;
it never prints credential values. OAuth failures omit token endpoint bodies,
authorization codes, PKCE verifiers and secret values.

Protocol decisions and pinned upstream sources are recorded in
[ADR 0007](adr/0007-canvas-oauth.md). Mock-server tests cover this flow.
Real institution acceptance remains tracked in
[#16](https://github.com/thedavidweng/canvas-cli/issues/16): an enabled Developer
Key, exact registered redirect URI, institutional approval and a real Canvas user
are still required to validate browser consent and real account reads.

## Auth commands

```bash
canvas auth login --base-url URL --token-stdin
canvas auth login --base-url URL --token-env CANVAS_TOKEN
canvas auth login --base-url URL --cookie-stdin
canvas auth login --base-url URL --cookie-env CANVAS_COOKIE
canvas auth login --base-url URL --cookie-file /path/to/cookie
canvas auth login --browser firefox
canvas auth login
canvas auth oauth login --login-timeout 5m
canvas auth refresh
canvas auth status
canvas auth test
canvas auth logout
canvas auth profiles
canvas auth use PROFILE
```

`auth status` shows credential presence and OAuth expiry but never credential values.

`auth test` calls the current user endpoint and returns clear JSON in `--json` mode. Works with token, OAuth and cookie auth.
