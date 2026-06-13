# Agent Workflow Examples

## Pull course context

```bash
canvas courses export-context --course 123 --json --out course-context.json
```

Agent use:

- parse `course-context.json`
- find assignments due soon
- inspect modules and readings
- plan work

## List unsubmitted assignments

```bash
canvas assignments list --course 123 --bucket unsubmitted --json
```

Parse:

```bash
jq '.data[] | {id, name, due_at, submission_types}'
```

## Dry-run an assignment submission

```bash
canvas assignments submit --course 123 456 --file paper.pdf --dry-run --json
```

Then confirm:

```bash
canvas assignments submit --course 123 456 --file paper.pdf --confirm --json
```

## Teaching team: download submissions

```bash
canvas submissions download --course 123 --assignment 456 --out submissions --json
```

Read manifest:

```bash
jq '.items[] | {user_id, filename, local_path, status}' submissions/manifest.json
```

## Teaching team: grade import

```bash
canvas grade import --course 123 --assignment 456 --csv grades.csv --dry-run --json
canvas grade import --course 123 --assignment 456 --csv grades.csv --confirm --json
```
