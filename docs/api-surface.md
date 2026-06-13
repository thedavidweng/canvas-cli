# Canvas API Surface

> [!IMPORTANT]
> For each command in [COMMANDS.md](../COMMANDS.md), this document lists the exact Canvas API endpoint, HTTP method, required query parameters, request body shape, and response shape. If an endpoint is missing here, consult the [Canvas LMS REST API documentation](https://canvas.instructure.com/doc/api/).

This document maps the CLI surface to Canvas API resource areas.

## Foundation APIs

| Area | Purpose | Notes |
|---|---|---|
| Users / Self | `canvas me get`, auth test | Use current user endpoint for connectivity validation. |
| Courses | course discovery and details | Student and teaching-team workflows start here. |
| Enrollments | role and section context | Needed for permissions, roster, teaching team workflows. |
| Sections | course organization | Needed for filtering students/submissions. |
| Modules | course structure | Primary course-context extraction path. |
| Assignments | due work and submissions | Core student and teaching workflow. |
| Files | resources and uploads | Needed for downloads, submissions, and course materials. |
| Announcements | course updates | Often represented as discussion topics. |
| Discussion Topics | forums and replies | Needed for student participation and teaching workflows. |
| Conversations | Canvas inbox | Needed for message read/reply/send. |
| Submissions | student submissions and grading | Needs student and teaching team paths. |
| Pages | course content | Needed to replace web UI for course browsing. |
| Calendar Events | due dates and schedule | Useful for agent planning. |
| Tabs | course navigation | Helps discover enabled tools in a course. |
| Rubrics | grading context | Needed for grading and feedback. |
| Assignment Groups | organization | Needed for gradebook-like context. |
| Groups | group assignments/discussions | Important for many courses. |

## Student workflows

### Pull full course context

Primary command:

```bash
canvas courses export-context --course COURSE_ID --out context.json
```

This should aggregate:

- course details
- tabs
- modules
- module items
- assignments
- assignment groups
- files metadata
- pages metadata/content when allowed
- announcements
- discussions
- calendar events
- user's submissions
- grades where available

The command should support:

```bash
--include files,pages,discussions,announcements,grades,submissions
--since DATE
--json
--out FILE
--download-files DIR
```

### Submit assignment

Commands:

```bash
canvas assignments submit --course COURSE_ID ASSIGNMENT_ID --text BODY
canvas assignments submit --course COURSE_ID ASSIGNMENT_ID --url URL
canvas assignments submit --course COURSE_ID ASSIGNMENT_ID --file PATH
canvas assignments submit --course COURSE_ID ASSIGNMENT_ID --file PATH --comment "See attached"
```

Implementation depends on Canvas submission type support. The command must inspect assignment submission types and reject invalid submission forms before sending.

### Discussion participation

Commands:

```bash
canvas discussions list --course COURSE_ID
canvas discussions get --course COURSE_ID DISCUSSION_ID
canvas discussions entries --course COURSE_ID DISCUSSION_ID
canvas discussions reply --course COURSE_ID DISCUSSION_ID --message BODY
canvas discussions reply-entry --course COURSE_ID DISCUSSION_ID --entry ENTRY_ID --message BODY
```

### Inbox

Commands:

```bash
canvas inbox list
canvas inbox get CONVERSATION_ID
canvas inbox send --to USER_ID --subject SUBJECT --body BODY
canvas inbox reply CONVERSATION_ID --body BODY
canvas inbox archive CONVERSATION_ID
```

## Teaching team workflows

### Roster and sections

```bash
canvas enrollments list --course COURSE_ID
canvas sections list --course COURSE_ID
canvas users list --course COURSE_ID
```

### Submissions and bulk download

```bash
canvas submissions list --course COURSE_ID --assignment ASSIGNMENT_ID
canvas submissions get --course COURSE_ID --assignment ASSIGNMENT_ID --user USER_ID
canvas submissions download --course COURSE_ID --assignment ASSIGNMENT_ID --out DIR
canvas submissions download --course COURSE_ID --assignment ASSIGNMENT_ID --section SECTION_ID --out DIR
```

Downloaded files should have deterministic names:

```text
<assignment-id>/<sortable-user-name>_<user-id>/<submission-id>_<original-filename>
```

Emit a manifest:

```text
manifest.json
manifest.ndjson
```

### Grading

```bash
canvas grade set --course COURSE_ID --assignment ASSIGNMENT_ID --user USER_ID --score SCORE
canvas grade comment --course COURSE_ID --assignment ASSIGNMENT_ID --user USER_ID --comment TEXT
canvas grade rubric --course COURSE_ID --assignment ASSIGNMENT_ID --user USER_ID --rubric-json FILE --dry-run
canvas grade rubric --course COURSE_ID --assignment ASSIGNMENT_ID --user USER_ID --rubric-json FILE --confirm
canvas grade import --course COURSE_ID --assignment ASSIGNMENT_ID --csv grades.csv --dry-run
canvas grade import --course COURSE_ID --assignment ASSIGNMENT_ID --csv grades.csv --confirm
```

Bulk grading must require `--dry-run` preview first or `--confirm` with explicit operation count.

### Course content modification

```bash
canvas pages update --course COURSE_ID PAGE_URL --body-file page.html --dry-run
canvas assignments update --course COURSE_ID ASSIGNMENT_ID --due-at 2026-06-30T23:59:00Z --dry-run
canvas modules publish --course COURSE_ID MODULE_ID --dry-run
canvas announcements create --course COURSE_ID --title TITLE --body-file BODY.md --dry-run
```

Course modification commands go through safety gates and audit logging.

## Raw API coverage

```bash
canvas api get /api/v1/courses
canvas api post /api/v1/courses/123/discussion_topics --data @payload.json --dry-run
canvas api put /api/v1/courses/123/assignments/456 --data @payload.json --dry-run
canvas api delete /api/v1/courses/123/files/789 --confirm
```

Raw write operations go through safety gates.

## Detailed Command API Surface

This section details the API endpoints and contracts for every command in `COMMANDS.md`.

### `canvas version`

- **Phase:** Foundation
- **HTTP Method:** N/A
- **Endpoint URL:** `N/A`
- **Query params:** N/A
- **Request body:** N/A
- **Response shape:** N/A
- **Permission notes:** N/A
- **Safety level:** Read-only

### `canvas doctor`

- **Phase:** Foundation
- **HTTP Method:** N/A
- **Endpoint URL:** `N/A (Uses various tests)`
- **Query params:** N/A
- **Request body:** N/A
- **Response shape:** N/A
- **Permission notes:** N/A
- **Safety level:** Read-only

### `canvas completion`

- **Phase:** Foundation
- **HTTP Method:** N/A
- **Endpoint URL:** `N/A`
- **Query params:** N/A
- **Request body:** N/A
- **Response shape:** N/A
- **Permission notes:** N/A
- **Safety level:** Read-only

### `canvas auth login`

- **Phase:** Foundation
- **HTTP Method:** N/A
- **Endpoint URL:** `N/A`
- **Query params:** N/A
- **Request body:** N/A
- **Response shape:** N/A
- **Permission notes:** N/A
- **Safety level:** Safe write (Local config only)

### `canvas auth status`

- **Phase:** Foundation
- **HTTP Method:** N/A
- **Endpoint URL:** `N/A`
- **Query params:** N/A
- **Request body:** N/A
- **Response shape:** N/A
- **Permission notes:** N/A
- **Safety level:** Read-only

### `canvas auth test`

- **Phase:** Foundation
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/users/self`
- **Query params:** None
- **Request body:** None
- **Response shape:** `{ id, name, sortable_name, short_name, login_id }`
- **Permission notes:** Requires valid token
- **Safety level:** Read-only

### `canvas auth logout`

- **Phase:** Foundation
- **HTTP Method:** N/A
- **Endpoint URL:** `N/A`
- **Query params:** N/A
- **Request body:** N/A
- **Response shape:** N/A
- **Permission notes:** N/A
- **Safety level:** Safe write (Local config only)

### `canvas me get`

- **Phase:** Foundation
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/users/self`
- **Query params:** None
- **Request body:** None
- **Response shape:** `{ id, name, sortable_name, short_name, login_id }`
- **Permission notes:** Requires valid token
- **Safety level:** Read-only

### `canvas me activity`

- **Phase:** Foundation
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/users/self/activity_stream`
- **Query params:** None
- **Request body:** None
- **Response shape:** `[ { id, title, message, type, read_state, created_at, html_url } ]`
- **Permission notes:** Requires valid token
- **Safety level:** Read-only

### `canvas api get`

- **Phase:** Foundation
- **HTTP Method:** GET
- **Endpoint URL:** `User provided path (e.g. /api/v1/courses)`
- **Query params:** User provided
- **Request body:** None
- **Response shape:** Raw JSON from Canvas
- **Permission notes:** Depends on endpoint
- **Safety level:** Read-only

### `canvas courses list`

- **Phase:** Student read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/courses`
- **Query params:** `enrollment_state=active`, `per_page=100`
- **Request body:** None
- **Response shape:** `[ { id, name, course_code, workflow_state, enrollments } ]`
- **Permission notes:** Must be enrolled
- **Safety level:** Read-only

### `canvas courses get`

- **Phase:** Student read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/courses/{course_id}`
- **Query params:** `include[]=term`, `include[]=total_scores`
- **Request body:** None
- **Response shape:** `{ id, name, course_code, term: { name } }`
- **Permission notes:** Must be enrolled
- **Safety level:** Read-only

### `canvas courses tabs`

- **Phase:** Student read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/courses/{course_id}/tabs`
- **Query params:** None
- **Request body:** None
- **Response shape:** `[ { id, html_url, full_url, position, visibility, label, type } ]`
- **Permission notes:** Must be enrolled
- **Safety level:** Read-only

### `canvas courses export-context`

- **Phase:** Student read
- **HTTP Method:** GET
- **Endpoint URL:** `Multiple (see export-context-spec.md)`
- **Query params:** Varies by endpoint
- **Request body:** None
- **Response shape:** Aggregated JSON envelope
- **Permission notes:** Must be enrolled
- **Safety level:** Read-only

### `canvas modules list`

- **Phase:** Student read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/courses/{course_id}/modules`
- **Query params:** `include[]=items`, `include[]=content_details`, `per_page=100`
- **Request body:** None
- **Response shape:** `[ { id, name, position, items_count, items: [ ... ] } ]`
- **Permission notes:** Must be enrolled
- **Safety level:** Read-only

### `canvas modules items`

- **Phase:** Student read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/courses/{course_id}/modules/{module_id}/items`
- **Query params:** `include[]=content_details`, `per_page=100`
- **Request body:** None
- **Response shape:** `[ { id, title, position, type, module_id, content_id, url } ]`
- **Permission notes:** Must be enrolled
- **Safety level:** Read-only

### `canvas assignments list`

- **Phase:** Student read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/courses/{course_id}/assignments`
- **Query params:** `per_page=100`, `order_by=due_at`
- **Request body:** None
- **Response shape:** `[ { id, name, description, due_at, points_possible, submission_types } ]`
- **Permission notes:** Must be enrolled
- **Safety level:** Read-only

### `canvas assignments get`

- **Phase:** Student read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/courses/{course_id}/assignments/{assignment_id}`
- **Query params:** None
- **Request body:** None
- **Response shape:** `{ id, name, description, due_at, points_possible, submission_types }`
- **Permission notes:** Must be enrolled
- **Safety level:** Read-only

### `canvas assignments groups`

- **Phase:** Student read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/courses/{course_id}/assignment_groups`
- **Query params:** `per_page=100`, `include[]=assignments`
- **Request body:** None
- **Response shape:** `[ { id, name, position, group_weight, assignments: [ ... ] } ]`
- **Permission notes:** Must be enrolled
- **Safety level:** Read-only

### `canvas files list`

- **Phase:** Student read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/courses/{course_id}/files`
- **Query params:** `per_page=100`, `sort=name`, `order=asc`
- **Request body:** None
- **Response shape:** `[ { id, display_name, size, url, content-type, created_at } ]`
- **Permission notes:** Must be enrolled
- **Safety level:** Read-only

### `canvas files download`

- **Phase:** Student read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/files/{file_id}`
- **Query params:** None
- **Request body:** None
- **Response shape:** `{ id, url, ... }` (Follows `url` to download file content)
- **Permission notes:** Must be enrolled and file must be visible
- **Safety level:** Read-only

### `canvas announcements list`

- **Phase:** Student read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/announcements`
- **Query params:** `context_codes[]=course_{course_id}`, `per_page=100`
- **Request body:** None
- **Response shape:** `[ { id, title, message, posted_at, read_state } ]`
- **Permission notes:** Must be enrolled
- **Safety level:** Read-only

### `canvas discussions list`

- **Phase:** Student read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/courses/{course_id}/discussion_topics`
- **Query params:** `only_announcements=false`, `per_page=100`, `order_by=recent_activity`
- **Request body:** None
- **Response shape:** `[ { id, title, message, discussion_type, read_state } ]`
- **Permission notes:** Must be enrolled
- **Safety level:** Read-only

### `canvas discussions get`

- **Phase:** Student read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/courses/{course_id}/discussion_topics/{discussion_id}`
- **Query params:** None
- **Request body:** None
- **Response shape:** `{ id, title, message, discussion_type }`
- **Permission notes:** Must be enrolled
- **Safety level:** Read-only

### `canvas pages list`

- **Phase:** Student read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/courses/{course_id}/pages`
- **Query params:** `per_page=100`, `sort=title`, `order=asc`
- **Request body:** None
- **Response shape:** `[ { url, title, created_at, updated_at, published } ]`
- **Permission notes:** Must be enrolled
- **Safety level:** Read-only

### `canvas pages get`

- **Phase:** Student read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/courses/{course_id}/pages/{url}`
- **Query params:** None
- **Request body:** None
- **Response shape:** `{ url, title, body, created_at, updated_at }`
- **Permission notes:** Must be enrolled
- **Safety level:** Read-only

### `canvas submissions get`

- **Phase:** Student read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/courses/{course_id}/assignments/{assignment_id}/submissions/{user_id}`
- **Query params:** None
- **Request body:** None
- **Response shape:** `{ id, body, url, grade, score, submitted_at, late }`
- **Permission notes:** Can only read own submission (unless teaching team)
- **Safety level:** Read-only

### `canvas assignments submit`

- **Phase:** Student actions
- **HTTP Method:** POST
- **Endpoint URL:** `/api/v1/courses/{course_id}/assignments/{assignment_id}/submissions`
- **Query params:** None
- **Request body:** `{ submission: { submission_type: 'online_text_entry'|'online_url'|'online_upload', body: '...', url: '...', file_ids: [...] } }`
- **Response shape:** `{ id, body, url, grade, score, submitted_at }`
- **Permission notes:** Assignment must be accepting submissions
- **Safety level:** Safe write

### `canvas discussions reply`

- **Phase:** Student actions
- **HTTP Method:** POST
- **Endpoint URL:** `/api/v1/courses/{course_id}/discussion_topics/{discussion_id}/entries`
- **Query params:** None
- **Request body:** `{ message: '...' }`
- **Response shape:** `{ id, user_id, message, created_at }`
- **Permission notes:** Discussion must be open and user enrolled
- **Safety level:** Safe write

### `canvas inbox list`

- **Phase:** Student actions
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/conversations`
- **Query params:** `per_page=100`
- **Request body:** None
- **Response shape:** `[ { id, subject, workflow_state, last_message, participants } ]`
- **Permission notes:** Valid token
- **Safety level:** Read-only

### `canvas inbox get`

- **Phase:** Student actions
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/conversations/{conversation_id}`
- **Query params:** None
- **Request body:** None
- **Response shape:** `{ id, subject, messages: [ ... ], participants }`
- **Permission notes:** Must be participant
- **Safety level:** Read-only

### `canvas inbox send`

- **Phase:** Student actions
- **HTTP Method:** POST
- **Endpoint URL:** `/api/v1/conversations`
- **Query params:** None
- **Request body:** `{ recipients: ['...'], subject: '...', body: '...' }`
- **Response shape:** `[ { id, subject, workflow_state, messages } ]`
- **Permission notes:** Valid token
- **Safety level:** Safe write

### `canvas inbox reply`

- **Phase:** Student actions
- **HTTP Method:** POST
- **Endpoint URL:** `/api/v1/conversations/{conversation_id}/add_message`
- **Query params:** None
- **Request body:** `{ body: '...' }`
- **Response shape:** `{ id, subject, messages: [ ... ] }`
- **Permission notes:** Must be participant
- **Safety level:** Safe write

### `canvas enrollments list`

- **Phase:** Teaching team read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/courses/{course_id}/enrollments`
- **Query params:** `per_page=100`, `include[]=total_scores`
- **Request body:** None
- **Response shape:** `[ { id, user_id, role, grades, user: { ... } } ]`
- **Permission notes:** Requires teaching/admin role
- **Safety level:** Read-only

### `canvas sections list`

- **Phase:** Teaching team read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/courses/{course_id}/sections`
- **Query params:** `per_page=100`, `include[]=total_students`
- **Request body:** None
- **Response shape:** `[ { id, name, course_id, total_students } ]`
- **Permission notes:** Requires teaching/admin role
- **Safety level:** Read-only

### `canvas users list`

- **Phase:** Teaching team read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/courses/{course_id}/users`
- **Query params:** `per_page=100`, `enrollment_type[]=student`
- **Request body:** None
- **Response shape:** `[ { id, name, sortable_name, short_name, login_id } ]`
- **Permission notes:** Requires teaching/admin role
- **Safety level:** Read-only

### `canvas submissions list`

- **Phase:** Teaching team read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/courses/{course_id}/assignments/{assignment_id}/submissions`
- **Query params:** `per_page=100`, `include[]=user`
- **Request body:** None
- **Response shape:** `[ { id, user_id, grade, score, submitted_at, workflow_state } ]`
- **Permission notes:** Requires teaching/admin role
- **Safety level:** Read-only

### `canvas submissions download`

- **Phase:** Teaching team read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/courses/{course_id}/assignments/{assignment_id}/submissions`
- **Query params:** `per_page=100` (followed by file downloads)
- **Request body:** None
- **Response shape:** Downloads files and creates manifest
- **Permission notes:** Requires teaching/admin role
- **Safety level:** Read-only

### `canvas rubrics list`

- **Phase:** Teaching team read
- **HTTP Method:** GET
- **Endpoint URL:** `/api/v1/courses/{course_id}/rubrics`
- **Query params:** `per_page=100`
- **Request body:** None
- **Response shape:** `[ { id, title, data: [ { id, description, points } ] } ]`
- **Permission notes:** Requires teaching/admin role
- **Safety level:** Read-only

### `canvas grade set`

- **Phase:** Teaching team actions
- **HTTP Method:** PUT
- **Endpoint URL:** `/api/v1/courses/{course_id}/assignments/{assignment_id}/submissions/{user_id}`
- **Query params:** None
- **Request body:** `{ submission: { posted_grade: '...' } }`
- **Response shape:** `{ id, grade, score, ... }`
- **Permission notes:** Requires teaching role
- **Safety level:** Destructive (requires confirm/dry-run in bulk)

### `canvas grade comment`

- **Phase:** Teaching team actions
- **HTTP Method:** PUT
- **Endpoint URL:** `/api/v1/courses/{course_id}/assignments/{assignment_id}/submissions/{user_id}`
- **Query params:** None
- **Request body:** `{ comment: { text_comment: '...' } }`
- **Response shape:** `{ id, submission_comments: [ ... ] }`
- **Permission notes:** Requires teaching role
- **Safety level:** Safe write

### `canvas grade rubric`

- **Phase:** Teaching team actions
- **HTTP Method:** GET / PUT
- **Endpoint URL:** `GET /api/v1/courses/{course_id}/rubrics/{rubric_id}` and `PUT /api/v1/courses/{course_id}/assignments/{assignment_id}/submissions/{user_id}`
- **Query params:** None
- **Request body:** 
  ```json
  {
    "rubric_assessment": {
      "criterion_id_1": {
        "points": 5,
        "comments": "Good thesis."
      }
    }
  }
  ```
- **Response shape:** `{ id, rubric_assessment: { ... } }`
- **Permission notes:** Requires teaching role
- **Safety level:** Destructive

### `canvas grade import`

- **Phase:** Teaching team actions
- **HTTP Method:** POST
- **Endpoint URL:** `/api/v1/courses/{course_id}/assignments/{assignment_id}/submissions/update_grades`
- **Query params:** None
- **Request body:** `{ grade_data: { 'user_id': { posted_grade: '...' } } }`
- **Response shape:** `[ { id, grade, ... } ]`
- **Permission notes:** Requires teaching role
- **Safety level:** Destructive (requires confirm/dry-run)

### `canvas announcements create`

- **Phase:** Teaching team actions
- **HTTP Method:** POST
- **Endpoint URL:** `/api/v1/courses/{course_id}/discussion_topics`
- **Query params:** None
- **Request body:** `{ is_announcement: true, title: '...', message: '...' }`
- **Response shape:** `{ id, title, message, ... }`
- **Permission notes:** Requires teaching role
- **Safety level:** Safe write

### `canvas assignments update`

- **Phase:** Teaching team actions
- **HTTP Method:** PUT
- **Endpoint URL:** `/api/v1/courses/{course_id}/assignments/{assignment_id}`
- **Query params:** None
- **Request body:** `{ assignment: { due_at: '...' } }`
- **Response shape:** `{ id, due_at, ... }`
- **Permission notes:** Requires teaching role
- **Safety level:** Destructive

### `canvas pages update`

- **Phase:** Teaching team actions
- **HTTP Method:** PUT
- **Endpoint URL:** `/api/v1/courses/{course_id}/pages/{url}`
- **Query params:** None
- **Request body:** `{ wiki_page: { body: '...' } }`
- **Response shape:** `{ url, body, ... }`
- **Permission notes:** Requires teaching role
- **Safety level:** Destructive

### `canvas modules publish`

- **Phase:** Teaching team actions
- **HTTP Method:** PUT
- **Endpoint URL:** `/api/v1/courses/{course_id}/modules/{module_id}`
- **Query params:** None
- **Request body:** `{ module: { published: true } }`
- **Response shape:** `{ id, published: true, ... }`
- **Permission notes:** Requires teaching role
- **Safety level:** Safe write

### `canvas modules unpublish`

- **Phase:** Teaching team actions
- **HTTP Method:** PUT
- **Endpoint URL:** `/api/v1/courses/{course_id}/modules/{module_id}`
- **Query params:** None
- **Request body:** `{ module: { published: false } }`
- **Response shape:** `{ id, published: false, ... }`
- **Permission notes:** Requires teaching role
- **Safety level:** Destructive

