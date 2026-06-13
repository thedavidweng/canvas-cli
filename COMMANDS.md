# Commands

This file is the target command inventory. Implementation status is marked for each command.

For authentication, users should use one of the following methods (the `--token` flag is not supported to avoid tokens in shell history):

```bash
canvas auth login --base-url URL --token-stdin
canvas auth login --base-url URL --token-env CANVAS_TOKEN
```

All commands below are implemented.

## Foundation

### version [done]

```bash
canvas version
```

### doctor [done]

```bash
canvas doctor
canvas doctor --json
canvas doctor --timeout 5s
```

### completion [done]

```bash
canvas completion bash
canvas completion zsh
canvas completion fish
canvas completion powershell
```

### auth [done]

```bash
canvas auth status
canvas auth status --json
canvas auth test
canvas auth test --json
canvas auth login --base-url https://school.instructure.com --token-stdin
canvas auth login --base-url https://school.instructure.com --token-env CANVAS_TOKEN
canvas auth logout
canvas auth profiles
canvas auth profiles --json
canvas auth use PROFILE
```

### me [done]

```bash
# [done]
canvas me get
canvas me get --json

# [done]
canvas me activity
canvas me todo
canvas me upcoming
```

### api [done]

```bash
# [done]
canvas api get /api/v1/courses
canvas api get /api/v1/courses/123/assignments --paginate --json
canvas api get /api/v1/courses --query "include[]=term" --raw
canvas api get /api/v1/courses --paginate --limit 50 --page-size 25

# [done]
canvas api post /api/v1/... --data @payload.json --dry-run
canvas api put /api/v1/... --data @payload.json --dry-run
canvas api delete /api/v1/... --confirm
```

## Student read

### courses [done]

```bash
canvas courses list
canvas courses list --json
canvas courses list --enrollment-state active
canvas courses list --enrollment-type student
canvas courses list --state available
canvas courses list --search "Computer Science"
canvas courses get 123
canvas courses get 123 --json
canvas courses tabs --course 123
canvas courses tabs --course 123 --json
canvas courses export-context --course 123 --out context.json
canvas courses export-context --course 123 --json
canvas courses export-context --course 123 --include "course,assignments,modules"
canvas courses export-context --course 123 --since "2026-01-01T00:00:00Z"
```

### modules [done]

```bash
canvas modules list --course 123
canvas modules list --course 123 --json
canvas modules get --course 123 456
canvas modules get --course 123 456 --json
canvas modules items --course 123 --module 456
canvas modules items --course 123 --module 456 --json
```

### modules item [done]

```bash
canvas modules item --course 123 --module 456 789
```

### modules publish/unpublish [done]

```bash
canvas modules publish --course 123 456 --dry-run
canvas modules publish --course 123 456 --confirm
canvas modules unpublish --course 123 456 --dry-run
canvas modules unpublish --course 123 456 --confirm
```

### assignments [done]

```bash
canvas assignments list --course 123
canvas assignments list --course 123 --json
canvas assignments list --course 123 --bucket unsubmitted
canvas assignments list --course 123 --bucket upcoming --sort due_at --order asc
canvas assignments list --course 123 --due-before "2026-07-01T00:00:00Z"
canvas assignments list --course 123 --include-submission
canvas assignments get --course 123 456
canvas assignments get --course 123 456 --json
canvas assignments groups --course 123
canvas assignments groups --course 123 --json
```

### assignments submit [done]

```bash
canvas assignments submit --course 123 456 --text "My answer"
canvas assignments submit --course 123 456 --url "https://example.com/homework"
canvas assignments submit --course 123 456 --file paper.pdf --dry-run
canvas assignments submit --course 123 456 --file paper.pdf --confirm
canvas assignments submit --course 123 456 --file paper.pdf --confirm --json
```

### assignments update [done]

```bash
canvas assignments update --course 123 456 --due-at "2026-07-15T23:59:00Z" --dry-run
canvas assignments update --course 123 456 --due-at "2026-07-15T23:59:00Z" --confirm
```

### announcements [done]

```bash
canvas announcements list --course 123
canvas announcements list --course 123 --json
canvas announcements create --course 123 --title "Midterm Review" --body-file review.md --dry-run
canvas announcements create --course 123 --title "Midterm Review" --body-file review.md --confirm
```

### announcements get [done]

```bash
canvas announcements get 456
```

### discussions [done]

```bash
canvas discussions list --course 123
canvas discussions list --course 123 --json
canvas discussions get --course 123 456
canvas discussions get --course 123 456 --json
canvas discussions entries --course 123 456
canvas discussions entries --course 123 456 --json
canvas discussions reply --course 123 --did 456 --message "Great point!" --dry-run
canvas discussions reply --course 123 --did 456 --message "Great point!" --confirm
canvas discussions reply-entry --course 123 --did 456 --entry 789 --message "I agree" --dry-run
canvas discussions reply-entry --course 123 --did 456 --entry 789 --message "I agree" --confirm
```

### discussions create [done]

```bash
canvas discussions create --course 123 --title "Week 1 Discussion" --body-file prompt.md --dry-run
```

### files [done]

```bash
canvas files list --course 123
canvas files list --course 123 --json
canvas files download 789 --out syllabus.pdf
canvas files download 789 --out syllabus.pdf --no-overwrite
```

### files get [done]

```bash
canvas files get 789
```

### files download-course [done]

```bash
canvas files download-course --course 123 --out ./course-files
```

### files upload [done]

```bash
canvas files upload --course 123 --file notes.pdf --folder 456 --dry-run
```

### pages [done]

```bash
canvas pages list --course 123
canvas pages list --course 123 --json
canvas pages get --course 123 syllabus
canvas pages get --course 123 syllabus --json
canvas pages update --course 123 syllabus --body-file new-body.md --dry-run
canvas pages update --course 123 syllabus --body-file new-body.md --confirm
```

### submissions [done]

```bash
canvas submissions get --course 123 --assignment 456 --user 789
canvas submissions get --course 123 --assignment 456 --user self --json
canvas submissions list --course 123 --assignment 456
canvas submissions list --course 123 --assignment 456 --json
canvas submissions download --course 123 --assignment 456 --out ./submissions
canvas submissions download --course 123 --assignment 456 --out ./submissions --no-overwrite --json
```

### submissions comment [done]

```bash
canvas submissions comment --course 123 --assignment 456 --user 789 --comment "Good work!" --dry-run
```

## Student actions

### inbox [done]

```bash
canvas inbox list
canvas inbox list --json
canvas inbox get 123
canvas inbox get 123 --json
canvas inbox send --to 456 --subject "Question" --body "When is the exam?" --dry-run
canvas inbox send --to 456 --subject "Question" --body "When is the exam?" --confirm
canvas inbox reply 123 --body "Thanks for the info!" --dry-run
canvas inbox reply 123 --body "Thanks for the info!" --confirm
canvas inbox archive 123 --dry-run
canvas inbox archive 123 --confirm
```

## Teaching team read

### enrollments [done]

```bash
canvas enrollments list --course 123
canvas enrollments list --course 123 --json
```

### sections [done]

```bash
canvas sections list --course 123
canvas sections list --course 123 --json
```

### users [done]

```bash
canvas users list --course 123
canvas users list --course 123 --json
canvas users list --course 123 --enrollment-type teacher
```

### rubrics [done]

```bash
canvas rubrics list --course 123
canvas rubrics list --course 123 --json
```

## Teaching team actions

### grade set [done]

```bash
canvas grade set --course 123 --assignment 456 --user 789 --score 95 --dry-run
canvas grade set --course 123 --assignment 456 --user 789 --score 95 --confirm
canvas grade set --course 123 --assignment 456 --user 789 --score 95 --confirm --json
```

### grade comment [done]

```bash
canvas grade comment --course 123 --assignment 456 --user 789 --comment "Excellent analysis" --dry-run
canvas grade comment --course 123 --assignment 456 --user 789 --comment "Excellent analysis" --confirm
```

### grade import [done]

```bash
canvas grade import --course 123 --assignment 456 --csv grades.csv --dry-run
canvas grade import --course 123 --assignment 456 --csv grades.csv --confirm
canvas grade import --course 123 --assignment 456 --csv grades.csv --confirm --json
canvas grade import --course 123 --assignment 456 --csv grades.csv --confirm --continue-on-error
```

### grade rubric [done]

```bash
canvas grade rubric --course 123 --assignment 456 --user 789 --rubric-json rubric.json --dry-run
canvas grade rubric --course 123 --assignment 456 --user 789 --rubric-json rubric.json --confirm
```
