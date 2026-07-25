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
- Exit code 7 identifies safety blocks (read-only, or a missing `--confirm`) for scripting.

## Amendment (2026-07-25): three-tier model

The four-tier table above was aspirational. No command ever constructed the
`Destructive` tier and no `delete` subcommand exists, so the `Destructive`
level, the `ErrNeedsConfirmDelete` error, `Policy.ConfirmDelete`, and the
`--confirm-delete` flag were dead code and have been removed. The implemented
model has three levels:

| Level | Examples | Gate |
|---|---|---|
| Read | list, get, download | Always allowed |
| Low-risk write | submit, reply, inbox send | `--confirm` |
| High-risk write | set grades, import, update, publish, edit, raw writes | `--confirm` |

`--dry-run` is always allowed and previews without sending. `--read-only`
(or `CANVAS_READ_ONLY=1`) blocks every write, overriding `--confirm`. Both a
read-only block and a missing `--confirm` exit with code 7. If a real delete
command is added later, reintroduce a destructive tier in code and amend this
ADR rather than resurrecting the removed flag speculatively.
