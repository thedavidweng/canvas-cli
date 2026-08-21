# Check what's due

Use this when you want a fast answer to "what do I need to work on?" — across all courses or scoped to one course — without opening the Canvas web UI.

> Outputs in the guides were captured from real runs of `canvas-cli` against a Canvas instance for a fictional student, *Alex Rivera*. IDs, names, and dates will differ in your account.

## Prerequisites

A working connection: `canvas auth test` succeeds. If not, start with [Getting started](../tutorials/getting-started.md).

## 1. Check your personal todo feed

```shell
canvas me todo
```

```text
[submitting] Problem Set 4: Recursion (due: 2026-08-28T23:59:00Z)
[submitting] Reading Quiz 6: Graphs (due: 2026-08-21T23:59:00Z)
[submitting] Lab Report: Sorting Algorithms (due: 2026-08-25T23:59:00Z)
```

This is your Canvas dashboard todo list: items you still need to submit, across all courses.

For calendar events (lectures, exams) rather than submission items:

```shell
canvas me upcoming
```

```text
[event] CS 101 Lecture: Dynamic Programming
  Start: 2026-08-24T15:00:00Z
  End:   2026-08-24T16:15:00Z
[event] MATH 210 Midterm Exam
  Start: 2026-09-14T14:00:00Z
  End:   2026-09-14T15:30:00Z
```

## 2. List unsubmitted work in one course

The todo feed spans courses. To drill into one course, filter its assignment list by bucket:

```shell
canvas assignments list --course 101 --bucket unsubmitted
```

```text
ID   Name                              Due At                Points  Published
---  --------------------------------  --------------------  ------  ---------
202  Problem Set 4: Recursion          2026-08-28T23:59:00Z  50      yes
203  Reading Quiz 6: Graphs            2026-08-21T23:59:00Z  10      yes
204  Lab Report: Sorting Algorithms    2026-08-25T23:59:00Z  25      yes
205  Project Showcase: Portfolio Link  2026-09-04T23:59:00Z  40      yes
```

Available buckets: `past`, `overdue`, `undated`, `ungraded`, `unsubmitted`, `upcoming`, `future`. For example, `--bucket overdue` shows only past-due work you haven't submitted.

## 3. Narrow by date or name

Everything due before a date:

```shell
canvas assignments list --course 101 --due-before 2026-08-26
```

```text
ID   Name                            Due At                Points  Published
---  ------------------------------  --------------------  ------  ---------
201  Problem Set 3: Lists and Trees  2026-08-14T23:59:00Z  50      yes
203  Reading Quiz 6: Graphs          2026-08-21T23:59:00Z  10      yes
204  Lab Report: Sorting Algorithms  2026-08-25T23:59:00Z  25      yes
```

Or search by name:

```shell
canvas assignments list --course 101 --search recursion
```

```text
ID   Name                      Due At                Points  Published
---  ------------------------  --------------------  ------  ---------
202  Problem Set 4: Recursion  2026-08-28T23:59:00Z  50      yes
```

Combine flags freely (`--bucket unsubmitted --sort due_at --order asc`).

## Next steps

- [Submit an assignment](submit-an-assignment.md) for one of the items you found
- [Check the JSON contract](../json-contract.md) if you want to script this (`--json`)
