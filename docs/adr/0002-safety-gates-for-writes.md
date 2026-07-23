# ADR-0002: Safety gates for write operations

## Status

Accepted

## Context

`canvas-cli` performs real Canvas writes: assignment submissions, discussion replies, inbox messages, grading, content edits, and raw API calls. These operations can affect grades, course content, and student data. Accidental or unintended writes are a significant risk, especially for automation agents running unattended.

## Decision

Every write command passes through a `safety.Policy` check before sending any request. The policy enforces a four-level safety model:

| Level | Examples | Gate |
|---|---|---|
| Read | list, get, download | Always allowed |
| Low-risk write | submit assignment, discussion reply, inbox send | `--confirm` |
| High-risk write | set grades, grade import, update due dates, publish/unpublish, edit pages | `--confirm` |
| Destructive | delete page, delete discussion, delete file, delete module item | `--confirm-delete` |

Global flags:

- `--dry-run` — always allowed, even under `--read-only`. Shows the intended mutation without sending it.
- `--confirm` — permits low-risk and high-risk writes.
- `--confirm-delete` — permits destructive operations.
- `--read-only` / `CANVAS_READ_ONLY=1` — blocks all writes, overrides `--confirm`. Exit code 7.

`CANVAS_READ_ONLY` is read at config resolution time, not in `Policy.Check`. The policy only sees the resolved `ReadOnly` boolean. This prevents env vars from bypassing the policy layer.

## Consequences

- No write command can execute without an explicit confirmation flag.
- `--read-only` is a hard stop for CI and automation environments that should never write.
- `--dry-run` is the default exploration path for agents and humans.
- Exit code 7 uniquely identifies read-only blocks for scripting.
