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
- Cookie values are never displayed in `auth status`, `--json`, `--verbose`, `--debug`, or doctor output.
- Cross-origin redirects strip all auth headers (Cookie, Authorization, X-CSRF-Token).

## OAuth2

OAuth2 is planned for multi-user distribution. The following commands are under consideration:

```bash
canvas auth oauth login
canvas auth oauth callback
canvas auth refresh
```

OAuth tokens would use OS keychain storage when feasible.

## Auth commands

```bash
canvas auth login --base-url URL --token-stdin
canvas auth login --base-url URL --token-env CANVAS_TOKEN
canvas auth login --base-url URL --cookie-stdin
canvas auth login --base-url URL --cookie-env CANVAS_COOKIE
canvas auth login --base-url URL --cookie-file /path/to/cookie
canvas auth login --browser firefox
canvas auth login
canvas auth status
canvas auth test
canvas auth logout
canvas auth profiles
canvas auth use PROFILE
```

`auth status` shows token and cookie presence but never the values.

`auth test` calls the current user endpoint and returns clear JSON in `--json` mode. Works with both token and cookie auth.
