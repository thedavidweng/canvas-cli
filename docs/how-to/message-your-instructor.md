# Message your instructor

Use this to ask a question in your Canvas inbox from the terminal, with a preview before anything is sent.

> Outputs in the guides were captured from real runs of `canvas-cli` against a Canvas instance for a fictional student, *Alex Rivera*. IDs, names, and dates will differ in your account.

## Prerequisites

A working connection (`canvas auth test` succeeds — see [Getting started](../tutorials/getting-started.md)).

## 1. Find the conversation or the recipient

To reply to an existing thread, list your inbox:

```shell
canvas inbox list
```

```text
401	unread	Question about Problem Set 3
402	read	Welcome to CS 101
```

Columns: conversation ID, read state, subject. Reply to it later with the ID from the first column.

To start a new thread you need the recipient's user ID. The course roster shows it for teaching staff:

```shell
canvas enrollments list --course 101
```

```text
Name          Role               Current Score  Current Grade
------------  -----------------  -------------  -------------
Dana Chen     TeacherEnrollment
Jamie Park    StudentEnrollment  92.5           A-
Sam Okafor    StudentEnrollment  92.5           A-
Lena Fischer  StudentEnrollment  92.5           A-
Alex Rivera   StudentEnrollment  88.0           B+
```

The roster doesn't print IDs in the table — run it with `--json` and read `user.id` from the envelope's `data` array.

## 2. Preview the message

```shell
canvas inbox send --to 60001 --subject "Question about Problem Set 4" \
  --body "Could you clarify what base case to use for exercise 2?" --dry-run
```

```text
DRY RUN: POST /api/v1/conversations
Payload: to=60001 subject="Question about Problem Set 4" body=Could you clarify what base case to use for exercise 2?
No mutation sent.
```

## 3. Send it

```shell
canvas inbox send --to 60001 --subject "Question about Problem Set 4" \
  --body "Could you clarify what base case to use for exercise 2?" --confirm
```

```text
Message sent (conversation 403)
```

## Replying to an existing conversation

Same gates, using the conversation ID from step 1:

```shell
canvas inbox reply 401 --body "Thanks — that clears it up!" --dry-run
canvas inbox reply 401 --body "Thanks — that clears it up!" --confirm
```

```text
Reply sent
```

## Next steps

- [Safety model](../safety-model.md) — what the gates guarantee, and the local audit log every send appends to
- [Command Reference](../command-spec.md#inbox) — archiving and other inbox commands
