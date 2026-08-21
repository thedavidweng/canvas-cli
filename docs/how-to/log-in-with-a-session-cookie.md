# Log in with a session cookie

Some schools disable personal access tokens in Canvas. If `canvas auth login` can't get you a working token, cookie auth lets the CLI use your browser's logged-in Canvas session instead.

> **Experimental and fallback-only.** Use token auth when your school allows it. A session cookie grants full account access — treat it like a password.

> Outputs in the guides were captured from real runs of `canvas-cli` against a Canvas instance for a fictional student, *Alex Rivera*. IDs, names, and config paths will differ in your account.

## How it works

`canvas auth login` reads the Canvas session cookie from your browser (auto-detected: Chrome, Firefox, Safari, Edge, Brave, Opera). Two consequences:

- **Cookies expire.** When your browser session ends — or after Canvas's idle timeout, typically 8–24 hours — you re-authenticate. Tokens don't have this problem.
- **Writes need a CSRF token.** Without one, read commands work but writes (submissions, messages, replies) fail. See step 3.

## 1. Log in interactively

```shell
canvas auth login
```

Choose **Session cookie (experimental)** when the wizard asks for the auth method. The CLI extracts the `_instructure_session`/`canvas_session` cookie — and the `_csrf_token` cookie for writes when present — from your browser automatically. If auto-detection picks the wrong browser, override it:

```shell
canvas auth login --browser firefox
```

If extraction fails entirely, choose the manual entry option and paste the cookie copied from your browser's DevTools (**Application → Cookies → your Canvas domain**).

## 2. Log in from scripts

For non-interactive use, pass the cookie over stdin, an environment variable, or a `0600` file — never a command-line flag:

```shell
echo "$CANVAS_COOKIE" | canvas auth login --base-url https://school.instructure.com --cookie-stdin
canvas auth login --base-url https://school.instructure.com --cookie-env CANVAS_COOKIE
canvas auth login --base-url https://school.instructure.com --cookie-file /path/to/cookie
```

Either path ends the same way — verify with:

```shell
canvas auth status
```

```text
Profile:    default
Base URL:   http://127.0.0.1:8787
Auth:       cookie (experimental)
```

Reads now work:

```shell
canvas courses list
```

```text
ID   Name                                 Code      State
---  -----------------------------------  --------  ---------
101  CS 101: Introduction to Programming  CS 101    available
102  MATH 210: Linear Algebra             MATH 210  available
103  HIST 140: Modern World History       HIST 140  available
```

Cookie values are stored with the same `0600` permissions as tokens, support `env:VAR_NAME` references in the config file, and are never displayed by `auth status`, `--json`, or `canvas doctor`.

## 3. Enable writes with a CSRF token

Under cookie auth, Canvas requires a CSRF token for writes. Without one:

```shell
canvas inbox send --to 60001 --subject t --body b --confirm
```

```text
Error: send message: request failed: csrf token required for mutation with cookie auth
```

To unblock writes, log in again passing the CSRF token alongside the cookie. In your browser's DevTools (**Application → Cookies → your Canvas domain**) it's the cookie named `_csrf_token`:

```shell
canvas auth login --base-url https://school.instructure.com --cookie-stdin --csrf-token-stdin
canvas auth login --base-url https://school.instructure.com --cookie-env CANVAS_COOKIE --csrf-token-env CANVAS_CSRF
```

## 4. When the session expires

Expired cookies fail with:

```text
Error: list courses: pagination failed: api error (status 401): session expired. Re-authenticate: canvas auth login
```

Run `canvas auth login` for a fresh cookie. If this happens constantly, ask your school's IT department about API token access — or accept that cookie auth is a re-login-every-few-hours workflow.

## Security notes

- Anyone with the cookie can act as you in Canvas — including SSO-protected sessions. Don't commit, share, or log it.
- Cross-origin redirects strip all auth headers (Cookie, Authorization, X-CSRF-Token), so your credentials don't leak to third-party hosts.
- OAuth2 for distributed multi-user apps is tracked in [issue #16](https://github.com/thedavidweng/canvas-cli/issues/16).

## Next steps

- [Authentication & Configuration](../auth.md) — config file layout, profiles, and precedence
- [Getting started](../tutorials/getting-started.md) — the token-based flow, if your school allows tokens
