# Product Brief: canvas-cli

`canvas-cli` is a local, single-binary Go CLI for Canvas LMS by Instructure. The repository is named `canvas-cli`; the installed binary is `canvas`.

Positioning:

> Canvas LMS CLI for automation agents, students, and teaching teams.

The project should eventually cover the main Canvas workflows that a student or teaching team uses in the browser:

- inspect courses
- pull modules, assignments, files, pages, announcements, discussions, calendar items, grades, and inbox conversations
- submit assignments
- participate in discussions
- send and reply to Canvas inbox messages
- download resources and submissions in bulk
- grade submissions where permissions allow
- comment on submissions
- update course content and assignment settings where permissions allow
- call any Canvas API endpoint directly via raw API commands

The product is not a toy Canvas wrapper. It is an automation substrate for agents such as OpenClaw, Hermes Agent, scripts, cron jobs, local developer tools, and human terminal users.

## Primary users

### Students

Students need to pull the full working context of a course quickly:

- all active courses
- course modules and module items
- assignments with due dates and submission status
- assignment details, rubrics, attachments, and submission requirements
- files, pages, syllabi, announcements, discussions, calendar events, grades
- inbox messages
- feedback from instructors and graders

Students also need safe actions:

- submit assignment text
- upload files for assignment submissions
- resubmit when Canvas permits it
- post discussion replies
- reply to inbox conversations
- mark/read inspect items for workflow automation

### Teaching team

Teachers, TAs, graders, and designers need operational workflows:

- list courses, sections, groups, enrollments, and users
- inspect assignments, rubrics, modules, pages, discussions, files, submissions, and gradebook-related data
- download submissions in bulk
- grade submissions and update scores/comments where permissions allow
- post announcements
- update assignment/module/page/discussion content where permissions allow
- batch edit due dates, availability dates, publishing state, and module structure
- export course context for offline review or agent-assisted workflows

### Agents

Agents need deterministic, parseable, reversible behavior:

- stable command names
- stable JSON output
- predictable exit codes
- stdout for data
- stderr for diagnostics
- non-interactive mode
- explicit confirmation gates for mutations
- local audit logs for every remote write
- raw API escape hatch

## Product principles

1. Reliability for automation comes first.
2. Stable JSON output is part of the public interface.
3. The CLI must match the owner's existing Go CLI house style.
4. Human table output is useful but secondary to machine-readable output.
5. Broad Canvas coverage is delivered through tested slices.
6. Write operations require strong safety gates and audit logs.
7. Raw API access unblocks workflows beyond first-class commands.
8. Canvas permissions must be surfaced clearly instead of hidden.

## Scope model

The project has two scopes:

- Foundation scope: transport, auth, config, output, pagination, rate limiting, errors, safety, audit, tests.
- Feature scope: Canvas resources and workflows.

Every feature must use the shared foundation. Avoid one-off HTTP code in commands.

## Non-negotiables

- Single binary.
- No hidden remote writes.
- No browser scraping.
- No TUI-first design.
- No token leakage.
- No unstable ad hoc JSON.
- No optimistic assumptions about user permissions.
- No aggressive concurrency by default.
