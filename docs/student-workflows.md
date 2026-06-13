# Student Workflows

Students can use `canvas` as a local command-line interface for their course work and as a structured context provider for agents.

## Pull course context

```bash
canvas courses list
canvas courses export-context --course 123 --out course-context.json
```

The exported context should include stable normalized objects and raw Canvas fields where useful.

JSON sections:

```json
{
  "course": {},
  "tabs": [],
  "modules": [],
  "module_items": [],
  "assignments": [],
  "submissions": [],
  "files": [],
  "pages": [],
  "announcements": [],
  "discussions": [],
  "calendar_events": [],
  "grades": []
}
```

Flags:

```text
--include modules,assignments,files,pages,announcements,discussions,calendar,grades,submissions
--since DATE
--download-files DIR
--out FILE
--json
```

## Track work

```bash
canvas assignments list --course 123 --bucket unsubmitted
canvas assignments list --course 123 --due-before 2026-06-30
canvas me todo
canvas me upcoming
```

## Inspect assignments

```bash
canvas assignments get --course 123 456 --json
canvas submissions get --course 123 --assignment 456 --user self --json
```

## Submit assignments

```bash
canvas assignments submit --course 123 456 --text "My answer" --dry-run
canvas assignments submit --course 123 456 --text "My answer" --confirm
canvas assignments submit --course 123 456 --file paper.pdf --dry-run
canvas assignments submit --course 123 456 --file paper.pdf --confirm
canvas assignments submit --course 123 456 --url https://example.com/project --confirm
```

Submission command requirements:

- inspect assignment submission types before submitting
- reject unsupported submission modes
- support dry-run
- show final submitted_at, workflow_state, and attempt if Canvas returns them
- upload files through the shared Canvas upload helper
- never silently overwrite local files

## Discussions

```bash
canvas discussions list --course 123
canvas discussions get --course 123 456
canvas discussions entries --course 123 456
canvas discussions reply --course 123 --did 456 --message "Great point!" --dry-run
canvas discussions reply --course 123 --did 456 --message "Great point!" --confirm
```

## Inbox

```bash
canvas inbox list
canvas inbox get 789
canvas inbox reply 789 --body "Thanks for the info!" --dry-run
canvas inbox reply 789 --body "Thanks for the info!" --confirm
canvas inbox send --to 12345 --subject "Question" --body "When is the exam?" --dry-run
```

## Download resources

```bash
canvas files list --course 123
canvas files download FILE_ID --out ./resources/file.pdf
canvas files download-course --course 123 --out ./course-files
```

Bulk downloads should write a manifest for agent use.
