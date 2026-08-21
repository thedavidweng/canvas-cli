# Documentation

`canvas-cli` documentation is organized by the [Diátaxis](https://diataxis.fr/) framework: tutorials teach, how-to guides solve, reference describes, explanation clarifies. Start where your goal is.

New to the CLI? Begin with [Getting started](tutorials/getting-started.md).

## Tutorials — learning-oriented

- **[Getting started](tutorials/getting-started.md)** — install, authenticate, and make your first queries in five minutes.

## How-to guides — task-oriented

For students:

- **[Check what's due](how-to/check-what-is-due.md)** — todo feed, buckets, due-date filters.
- **[Submit an assignment](how-to/submit-an-assignment.md)** — text, file, or URL, with a preview first.
- **[Download course files](how-to/download-course-files.md)** — one file or the whole course, with manifests.
- **[Message your instructor](how-to/message-your-instructor.md)** — inbox send and reply from the terminal.
- **[Log in with a session cookie](how-to/log-in-with-a-session-cookie.md)** — for schools that disable access tokens.

For teaching teams:

- **[Collect student submissions](how-to/collect-student-submissions.md)** — bulk download with a manifest.
- **[Enter grades](how-to/enter-grades.md)** — single scores, comments, and CSV import behind safety gates.

## Reference — information-oriented

- **[Command Reference](command-spec.md)** — every command, flag, and environment variable.
- **[JSON Contract](json-contract.md)** — the `--json` envelope schema and exit codes.
- **[Authentication & Configuration](auth.md)** — config file, profiles, precedence, token and cookie details.
- **[Raw API](raw-api.md)** — the `canvas api` escape hatch for untyped endpoints.
- **[API Surface](api-surface.md)** — Canvas endpoints the CLI exercises.

## Explanation — understanding-oriented

- **[Safety Model](safety-model.md)** — `--dry-run`, `--confirm`, `--read-only`, and the audit log.
- **[Architecture](architecture.md)** — codebase structure and design decisions.
- **[Architecture Decision Records](adr/)** — why things are the way they are.

## Project

- **[Contributing](../CONTRIBUTING.md)** — development setup and guidelines.
- **[Security](../SECURITY.md)** — reporting vulnerabilities.

All command examples in the tutorials and how-to guides were captured from real runs of the CLI; where an output is environment-specific (request IDs, timing), the article says so.
