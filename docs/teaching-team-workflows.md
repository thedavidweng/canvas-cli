# Teaching Team Workflows

Teaching team commands require careful permission handling and safety gates. The same binary supports teachers, TAs, graders, designers, and admins where Canvas permissions allow it.

## Course operations

```bash
canvas courses list --enrollment-type teacher
canvas courses get 123
canvas courses tabs --course 123
canvas courses export-context --course 123 --out course-context.json
```

## Roster

```bash
canvas enrollments list --course 123
canvas sections list --course 123
canvas users list --course 123
```

Roster commands support filtering by section, role, state, and search string.

## Assignment operations

```bash
canvas assignments list --course 123
canvas assignments get --course 123 456
canvas assignments update --course 123 456 --due-at 2026-06-30T23:59:00Z --dry-run
canvas assignments update --course 123 456 --published true --dry-run
```

High-risk assignment updates require `--confirm`.

## Bulk submission download

```bash
canvas submissions download --course 123 --assignment 456 --out submissions --dry-run
canvas submissions download --course 123 --assignment 456 --out submissions --confirm
canvas submissions download --course 123 --assignment 456 --section 789 --out submissions --confirm
```

Required output:

```text
submissions/
  manifest.json
  manifest.ndjson
  assignment-456/
    user-name_111/
      submission-222_original.pdf
```

Manifest should include:

- course_id
- assignment_id
- user_id
- submission_id
- attempt
- submitted_at
- late/missing/excused flags if available
- file_id
- original filename
- local path
- download status
- error if failed

## Grading

```bash
canvas grade set --course 123 --assignment 456 --user 111 --score 95 --dry-run
canvas grade set --course 123 --assignment 456 --user 111 --score 95 --confirm
canvas grade comment --course 123 --assignment 456 --user 111 --comment "Excellent analysis" --dry-run
canvas grade import --course 123 --assignment 456 --csv grades.csv --dry-run
canvas grade import --course 123 --assignment 456 --csv grades.csv --confirm
```

Grade import CSV columns:

```csv
user_id,score,comment
111,95,"Good work"
```

Rubric assessment is available via `canvas grade rubric`.

## Announcements and discussions

```bash
canvas announcements create --course 123 --title "Update" --body-file body.md --dry-run
canvas discussions create --course 123 --title "Week 4" --body-file prompt.md --dry-run
```

## Course content edits

```bash
canvas pages update --course 123 PAGE_URL --body-file page.html --dry-run
canvas modules publish --course 123 MODULE_ID --dry-run
canvas modules unpublish --course 123 MODULE_ID --dry-run
canvas files upload --course 123 --file syllabus.pdf --dry-run
```

These commands show a payload summary before mutation.
