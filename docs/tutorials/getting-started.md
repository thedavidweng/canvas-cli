# Getting started with canvas-cli

This tutorial takes you from zero to your first Canvas query: install the CLI, connect it to your school's Canvas instance, and read your own data from the terminal. It takes about five minutes.

> All outputs in the guides were captured from real runs of `canvas-cli` against a Canvas instance for a fictional student, *Alex Rivera*. IDs, names, URLs, request IDs, and timing values will differ in your account.

## What you'll do

1. Install the `canvas` binary
2. Connect it to your Canvas instance with an access token
3. Verify the connection
4. List your courses and your upcoming work
5. Preview a write before sending it

## Prerequisites

- A Canvas account at an institution that allows personal access tokens. If your school disables token generation, follow [Log in with a session cookie](../how-to/log-in-with-a-session-cookie.md) instead of step 2.
- A terminal.

## 1. Install

macOS or Linux:

```shell
curl -fsSL https://raw.githubusercontent.com/thedavidweng/canvas-cli/main/install.sh | sh
```

Windows (PowerShell):

```shell
powershell -ExecutionPolicy ByPass -c "irm https://raw.githubusercontent.com/thedavidweng/canvas-cli/main/install.ps1 | iex"
```

Homebrew, Go, and manual download options are listed in the [README](../../README.md). Confirm the binary is on your `PATH`:

```shell
canvas version
```

```text
canvas 0.4.0 (commit: ca7d863, built: 2026-08-18T21:23:04-07:00)
```

## 2. Connect to your Canvas instance

First create an access token in the Canvas web UI:

```text
Account -> Settings -> Approved Integrations -> New Access Token
```

Then run:

```shell
canvas auth login
```

The wizard asks for a profile name, your Canvas instance URL (for example `https://school.instructure.com`), and the token. It verifies the token against your instance and saves it to your config file with `0600` permissions.

Prefer not to paste secrets interactively (for example, in scripts)? Feed the token over stdin instead — never as a command-line flag, so it stays out of your shell history:

```shell
printf '%s' "$CANVAS_TOKEN" | canvas auth login --base-url https://school.instructure.com --token-stdin
```

## 3. Verify the connection

```shell
canvas auth test
```

```text
Authentication successful!
User:  Alex Rivera
ID:    61001
Login: alex.rivera
```

If this fails, run `canvas doctor` for a per-check diagnosis of config, token, base URL, and API reachability.

## 4. Read your courses and upcoming work

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

Every course command takes the course ID from the first column. To see what needs your attention:

```shell
canvas me todo
```

```text
[submitting] Problem Set 4: Recursion (due: 2026-08-28T23:59:00Z)
[submitting] Reading Quiz 6: Graphs (due: 2026-08-21T23:59:00Z)
[submitting] Lab Report: Sorting Algorithms (due: 2026-08-25T23:59:00Z)
```

## 5. Try JSON output

Every command accepts `--json` and emits a stable envelope — the contract scripts and agents can rely on:

```shell
canvas courses list --json --pretty
```

```json
{
  "ok": true,
  "data": [
    {
      "id": "101",
      "name": "CS 101: Introduction to Programming",
      "course_code": "CS 101",
      "workflow_state": "available",
      "enrollment_term_id": "1",
      "term": {
        "id": "1",
        "name": "Fall 2026"
      }
    },
    {
      "id": "102",
      "name": "MATH 210: Linear Algebra",
      "course_code": "MATH 210",
      "workflow_state": "available",
      "enrollment_term_id": "1",
      "term": {
        "id": "1",
        "name": "Fall 2026"
      }
    },
    {
      "id": "103",
      "name": "HIST 140: Modern World History",
      "course_code": "HIST 140",
      "workflow_state": "available",
      "enrollment_term_id": "1",
      "term": {
        "id": "1",
        "name": "Fall 2026"
      }
    }
  ],
  "meta": {
    "schema_version": "2026-07-25",
    "command": "courses.list",
    "request_id": "89cad9a5-37f3-473f-a837-5ac4a35afc09",
    "base_url": "http://127.0.0.1:8787",
    "duration_ms": 8,
    "limit": null
  }
}
```

`meta.request_id`, `meta.base_url`, and `meta.duration_ms` describe the individual request and change every run. The full envelope schema, exit codes, and error shapes are specified in the [JSON contract](../json-contract.md).

## 6. Preview a write before sending it

Commands that change anything in Canvas (submissions, messages, replies) support three safety gates: `--dry-run` to preview, `--confirm` to execute, and `--read-only` to hard-block. Try a preview — nothing is sent:

```shell
canvas inbox send --to 60001 --subject "Question about Problem Set 4" \
  --body "Could you clarify what base case to use for exercise 2?" --dry-run
```

```text
DRY RUN: POST /api/v1/conversations
Payload: to=60001 subject="Question about Problem Set 4" body=Could you clarify what base case to use for exercise 2?
No mutation sent.
```

When the preview looks right, re-run the same command with `--confirm` to send it. The [safety model](../safety-model.md) explains the gates, and [Message your instructor](../how-to/message-your-instructor.md) walks through the full send.

## Troubleshooting

**`api error (status 401): session expired. Re-authenticate: canvas auth login`** — your token or cookie is no longer valid. Run `canvas auth login` again.

**`config error: ... (run canvas auth login to set up credentials)`** — no credentials found. Complete step 2.

**Something else?** Run `canvas doctor` and check the failing row, or see [Authentication & Configuration](../auth.md).

## Next steps

- [Check what's due](../how-to/check-what-is-due.md) — buckets, due-date filters, and the todo feed
- [Submit an assignment](../how-to/submit-an-assignment.md)
- [Download course files](../how-to/download-course-files.md)
- [Command Reference](../command-spec.md) — every command and flag
