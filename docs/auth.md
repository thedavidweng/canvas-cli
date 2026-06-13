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
CANVAS_PROFILE=default
CANVAS_CONFIG=/path/to/config.yaml
```

`CANVAS_BASE_URL` is the Canvas root URL. Accept accidental `/api/v1` suffixes and normalize internally.

## Config file

Default path:

```text
${XDG_CONFIG_HOME:-~/.config}/canvas-cli/config.yaml
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
canvas auth login
canvas auth status
canvas auth test
canvas auth logout
canvas auth profiles
canvas auth use PROFILE
```

`auth status` should show token presence but never the token.

`auth test` should call the current user endpoint and return clear JSON in `--json` mode.
