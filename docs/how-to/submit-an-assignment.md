# Submit an assignment

Use this to turn in work from the terminal — as text, a file upload, or a URL — with a preview before anything is sent.

> Outputs in the guides were captured from real runs of `canvas-cli` against a Canvas instance for a fictional student, *Alex Rivera*. IDs, names, and dates will differ in your account.

## Prerequisites

A working connection (`canvas auth test` succeeds — see [Getting started](../tutorials/getting-started.md)) and the course plus assignment IDs (`canvas assignments list --course COURSE_ID`, or [Check what's due](check-what-is-due.md)).

## 1. Check what the assignment accepts

Assignments only accept certain submission types, and the CLI refuses mismatches. Look at the `Types:` line first:

```shell
canvas assignments get --course 101 202
```

```text
ID:        202
Name:      Problem Set 4: Recursion
Course ID: 101
Points:    50
Published: true
Due At:    2026-08-28T23:59:00Z
Types:     online_text_entry, online_upload
```

This assignment accepts text entries and file uploads. The type must also be one the CLI supports: `online_text_entry` (`--text`), `online_upload` (`--file`), or `online_url` (`--url`). Quizzes (`online_quiz`) must be taken in the Canvas web UI.

## 2. Preview the submission

Build the submission with `--dry-run` first. Nothing is sent:

```shell
canvas assignments submit --course 101 202 \
  --text "Exercise 1: ...base case is the empty list... Exercise 2: see attached reasoning." \
  --dry-run
```

```text
DRY RUN: POST /api/v1/courses/101/assignments/202/submissions
Resource IDs: 101, 202
Payload: type=online_text_entry body={"submission_type":"online_text_entry","body":"Exercise 1: ...base case is the empty list... Exercise 2: see attached re...
No mutation sent.
```

The preview shows the exact request that would be made. Long bodies are truncated in the preview only.

## 3. Submit

Re-run with `--confirm`:

```shell
canvas assignments submit --course 101 202 \
  --text "Exercise 1: ...base case is the empty list... Exercise 2: see attached reasoning." \
  --confirm
```

```text
Submission submitted (ID: 9001, state: submitted)
```

## 4. Verify what Canvas recorded

```shell
canvas submissions get --course 101 --assignment 202 --user self
```

```text
ID:             9001
User ID:        61001
Assignment ID:  202
State:          submitted
Submitted At:   2026-08-21T14:32:11Z
Late:           false
Missing:        false
```

`State: submitted` with a `Submitted At` timestamp confirms Canvas accepted it. Once graded, `Score` appears here too.

## Submitting a file instead

Same pattern — preview, then confirm. The file is uploaded through Canvas's upload flow, then attached to the submission:

```shell
canvas assignments submit --course 101 202 --file ./ps4-recursion.pdf --dry-run
canvas assignments submit --course 101 202 --file ./ps4-recursion.pdf --confirm
```

```text
Submission submitted (ID: 9001, state: submitted)
```

## Submitting a URL

For assignments whose `Types:` include `online_url`:

```shell
canvas assignments submit --course 101 205 --url https://github.com/alex-rivera/cs101-portfolio --confirm
```

```text
Submission submitted (ID: 9001, state: submitted)
```

## If the type doesn't match, the CLI stops you

Submitting a URL to an assignment that doesn't accept one fails before any request is made:

```shell
canvas assignments submit --course 101 202 --url https://example.com/demo --confirm
```

```text
Error: assignment 202 does not accept online_url submission (allowed: online_text_entry, online_upload)
```

## Next steps

- [Check what's due](check-what-is-due.md) — find the next thing to submit
- [Safety model](../safety-model.md) — how `--dry-run`, `--confirm`, and `--read-only` interact
- Set `CANVAS_READ_ONLY=1` in your shell to explore without any risk of writes
