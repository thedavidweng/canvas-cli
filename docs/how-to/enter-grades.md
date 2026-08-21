# Enter grades

Use this to record scores and feedback in Canvas — one student at a time or in bulk from a CSV — with a preview of every write before it happens.

> Outputs in the guides were captured from real runs of `canvas-cli` against a Canvas instance for a fictional teaching team. IDs, names, and scores will differ in your course.

## Prerequisites

A teacher/TA enrollment, a working connection (`canvas auth test`), and the assignment ID (`canvas assignments list --course COURSE_ID`).

Grading is a high-risk write: every command below previews by default with `--dry-run` and only mutates Canvas when you add `--confirm`.

## 1. Enter one score

Preview first:

```shell
canvas grade set --course 101 --assignment 202 --user 61003 --score 42 --dry-run
```

```text
DRY RUN: PUT /api/v1/courses/101/assignments/202/submissions/61003
Resource IDs: 101, 202, 61003
Payload: {"submission":{"posted_grade":"42"}}
No mutation sent.
```

Then send:

```shell
canvas grade set --course 101 --assignment 202 --user 61003 --score 42 --confirm
```

```text
Grade set to 42 for user 61003 on assignment 202
```

## 2. Attach feedback

```shell
canvas grade comment --course 101 --assignment 202 --user 61003 \
  --comment "Recovered — see extension email" --dry-run
```

```text
DRY RUN: PUT /api/v1/courses/101/assignments/202/submissions/61003
Resource IDs: 101, 202, 61003
Payload: {"comment":{"text_comment":"Recovered — see extension email"}}
No mutation sent.
```

```shell
canvas grade comment --course 101 --assignment 202 --user 61003 \
  --comment "Recovered — see extension email" --confirm
```

```text
Comment added for user 61003 on assignment 202
```

## 3. Import grades in bulk

Create a CSV with `user_id`, `score`, and an optional `comment` per row:

```csv
user_id,score,comment
61003,42,"Recovered — see extension email"
61004,45,"Late penalty applied (-5)"
```

Preview the whole import:

```shell
canvas grade import --course 101 --assignment 202 --csv grades.csv --dry-run
```

```text
DRY RUN: POST /api/v1/courses/101/assignments/202/submissions/update_grades
Resource IDs: 101, 202
Payload: 2 grades:
  user 61003 -> 42
  user 61004 -> 45

No mutation sent.
```

Then execute:

```shell
canvas grade import --course 101 --assignment 202 --csv grades.csv --confirm
```

```text
warning: importing 2 grades without prior --dry-run
Imported 2 grades for assignment 202
```

(The warning appears on every direct `--confirm` run as a reminder to preview first; running the `--dry-run` above beforehand is the intended workflow.)

## 4. Explore safely

Set read-only mode while you're learning the commands — writes are refused outright, even with `--confirm`:

```shell
CANVAS_READ_ONLY=1 canvas grade set --course 101 --assignment 202 --user 61003 --score 42 --confirm
```

```text
Error: operation blocked by read-only mode
```

When you enable the audit log in your config file (`audit:` → `enabled: true`), every executed grade change is also appended to a local JSONL file with the request path, a payload hash, and the response status — see [Safety model](../safety-model.md) for the schema and per-OS default path.

## Next steps

- [Collect student submissions](collect-student-submissions.md) — download the work you're grading
- [Safety model](../safety-model.md) — the full gate matrix and audit-log schema
- Rubric-based assessment: `canvas grade rubric --rubric-json FILE` (see [Command Reference](../command-spec.md#grading))
