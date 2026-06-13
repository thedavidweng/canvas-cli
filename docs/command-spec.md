# Command Specification

Binary name: `canvas`.

Global flags:

```text
--config PATH
--profile NAME
--base-url URL
--json
--pretty
--compact
--ndjson
--full
--limit N
--page-size N
--no-paginate
--timeout DURATION
--retries N
--dry-run
--confirm
--read-only
--events
--verbose
--debug
--quiet
--no-color
--confirm-delete
```

Environment variables:

```text
CANVAS_BASE_URL      # Canvas instance base URL
CANVAS_TOKEN         # API token (or env:VARNAME reference in config)
CANVAS_PROFILE       # Config profile name
CANVAS_READ_ONLY     # Set to "1" to block all write operations
```

## Root commands

```bash
canvas version
canvas doctor
canvas completion bash|zsh|fish|powershell
```

`doctor` should validate config, auth, base URL, token presence, API reachability, clock sanity if useful, and write-safety settings.

### `canvas doctor` output

`doctor` validates the CLI environment and reports each check:

| Check | Pass condition |
|---|---|
| Config file | File exists and is parseable YAML. If missing but env vars are sufficient, mark as SKIP or WARN (do not fail). |
| Config permissions | File has 0600 permissions (only checked if file exists) |
| Token present | Token resolved from env or config |
| Base URL | Base URL is set and is a valid HTTPS URL |
| API reachable | `GET /api/v1/users/self` returns 200 |
| Token valid | API call succeeds with current token |
| Write safety | `--read-only` / `CANVAS_READ_ONLY` status reported |

**Human output:** table of check name + status (✓ / ✗ / ⚠).

**JSON output:** array of check objects in `data`:
```json
{ "check": "api_reachable", "status": "pass", "message": "" }
```

**Exit codes:** 0 if all critical checks pass (missing config file is OK if env vars provide full config), 3 if auth/config fails, 6 if network fails.

## Auth

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

`auth login` may support interactive fallback for humans. Agents should use environment variables or explicit flags.
Note: the `--token` flag is not supported to avoid tokens in shell history. Users should use `--token-stdin` or `--token-env`.

## Me

```bash
canvas me get
canvas me activity
canvas me todo
canvas me upcoming
```

## Courses

```bash
canvas courses list
canvas courses get COURSE_ID
canvas courses tabs --course COURSE_ID
canvas courses export-context --course COURSE_ID --out context.json
```

Useful flags:

```text
--enrollment-state active|invited_or_pending|completed
--enrollment-type teacher|student|ta|observer|designer
--state available|completed|unpublished
--include term,total_scores,current_grading_period_scores,sections
--search TEXT
```

## Modules

```bash
canvas modules list --course COURSE_ID
canvas modules get --course COURSE_ID MODULE_ID
canvas modules items --course COURSE_ID --module MODULE_ID
canvas modules item --course COURSE_ID --module MODULE_ID ITEM_ID
canvas modules publish --course COURSE_ID MODULE_ID --dry-run
canvas modules unpublish --course COURSE_ID MODULE_ID --dry-run
```

## Assignments

```bash
canvas assignments list --course COURSE_ID
canvas assignments get --course COURSE_ID ASSIGNMENT_ID
canvas assignments groups --course COURSE_ID
canvas assignments submit --course COURSE_ID ASSIGNMENT_ID --text BODY
canvas assignments submit --course COURSE_ID ASSIGNMENT_ID --file PATH
canvas assignments submit --course COURSE_ID ASSIGNMENT_ID --url URL
canvas assignments update --course COURSE_ID ASSIGNMENT_ID --due-at TIME --dry-run
```

Useful list flags:

```text
--search TEXT
--bucket past|overdue|undated|ungraded|unsubmitted|upcoming|future
--due-before DATE
--due-after DATE
--published true|false
--include-submission
--sort due_at|name|position
--order asc|desc
```

## Announcements

```bash
canvas announcements list --course COURSE_ID
canvas announcements get --course COURSE_ID ANNOUNCEMENT_ID
canvas announcements create --course COURSE_ID --title TITLE --body-file FILE --dry-run
```

Useful flags:

```text
--start-date DATE
--end-date DATE
--active-only
--latest-only
```

## Discussions

```bash
canvas discussions list --course COURSE_ID
canvas discussions get --course COURSE_ID DISCUSSION_ID
canvas discussions entries --course COURSE_ID DISCUSSION_ID
canvas discussions reply --course COURSE_ID --did DISCUSSION_ID --message BODY --dry-run
canvas discussions reply-entry --course COURSE_ID --did DISCUSSION_ID --entry ENTRY_ID --message BODY --dry-run
canvas discussions create --course COURSE_ID --title TITLE --body-file BODY.md --dry-run
```

## Files

```bash
canvas files list --course COURSE_ID
canvas files get FILE_ID
canvas files download FILE_ID --out PATH
canvas files download-course --course COURSE_ID --out DIR
canvas files upload --course COURSE_ID --file PATH --folder FOLDER_ID --dry-run
```

Useful flags:

```text
--search TEXT
--content-type TYPE
--exclude-content-type TYPE
--sort name|size|created_at|updated_at|content_type|user
--order asc|desc
```

## Pages

```bash
canvas pages list --course COURSE_ID
canvas pages get --course COURSE_ID PAGE_URL
canvas pages update --course COURSE_ID PAGE_URL --body-file FILE --dry-run
```

## Inbox

```bash
canvas inbox list
canvas inbox get CONVERSATION_ID
canvas inbox send --to USER_ID --subject SUBJECT --body BODY --dry-run
canvas inbox reply CONVERSATION_ID --body BODY --dry-run
canvas inbox archive CONVERSATION_ID --dry-run
```

## Enrollments

```bash
canvas enrollments list --course COURSE_ID
```

## Sections

```bash
canvas sections list --course COURSE_ID
```

## Users

```bash
canvas users list --course COURSE_ID
```

## Rubrics

```bash
canvas rubrics list --course COURSE_ID
```

## Submissions
```bash
canvas submissions list --course COURSE_ID --assignment ASSIGNMENT_ID
canvas submissions get --course COURSE_ID --assignment ASSIGNMENT_ID --user USER_ID
canvas submissions download --course COURSE_ID --assignment ASSIGNMENT_ID --out DIR
canvas submissions comment --course COURSE_ID --assignment ASSIGNMENT_ID --user USER_ID --comment TEXT --dry-run
```

## Grading

```bash
canvas grade set --course COURSE_ID --assignment ASSIGNMENT_ID --user USER_ID --score SCORE --dry-run
canvas grade comment --course COURSE_ID --assignment ASSIGNMENT_ID --user USER_ID --comment TEXT --dry-run
canvas grade rubric --course COURSE_ID --assignment ASSIGNMENT_ID --user USER_ID --rubric-json FILE --dry-run
canvas grade rubric --course COURSE_ID --assignment ASSIGNMENT_ID --user USER_ID --rubric-json FILE --confirm

canvas grade import --course COURSE_ID --assignment ASSIGNMENT_ID --csv FILE --dry-run
```

## Raw API

```bash
canvas api get /api/v1/courses
canvas api get /api/v1/courses/123/assignments --paginate
canvas api post /api/v1/... --data @payload.json --dry-run
canvas api put /api/v1/... --data @payload.json --dry-run
canvas api delete /api/v1/... --confirm
```

Raw write methods honor all safety gates (dry-run, confirm, audit log).
