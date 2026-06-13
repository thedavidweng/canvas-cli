# Architecture

`canvas-cli` is a layered Go CLI. Commands stay thin. Canvas API behavior lives in shared packages.

## Package layout

```text
cmd/canvas/
  main.go
internal/cli/
  root.go
  auth.go
  me.go
  courses.go
  assignments.go
  modules.go
  files.go
  discussions.go
  inbox.go
  submissions.go
  grading.go
  pages.go
  api.go
internal/config/
  config.go
internal/canvas/
  client.go
  request.go
  pagination.go
  ratelimit.go
  errors.go
  uploads.go
  types.go
  courses.go
  assignments.go
  modules.go
  files.go
  submissions.go
  discussions.go
  conversations.go
  pages.go
  announcements.go
  grading.go
internal/output/
  envelope.go
  json.go
  table.go
  exitcode.go
internal/safety/
  policy.go
  dryrun.go
internal/audit/
  audit.go
internal/testutil/
  canvas_mock.go
```

## Dependency direction

```text
cmd/canvas -> internal/cli -> internal/canvas
                           -> internal/config
                           -> internal/output
                           -> internal/safety
                           -> internal/audit
```

`internal/canvas` must not depend on `internal/cli`.

## HTTP client

Use `net/http` and make the transport testable.

Required behavior:

- configurable base URL
- bearer token auth
- explicit user agent
- request timeout
- context cancellation
- default single-flight request concurrency
- automatic pagination helper
- Link header parser
- retry/backoff for throttling and transient failures
- error body preservation
- response metadata capture
- token redaction

Default headers:

```text
Authorization: Bearer <token>
User-Agent: canvas-cli/<version> (+https://github.com/thedavidweng/canvas-cli)
Accept: application/json+canvas-string-ids
```

Use Canvas string IDs by default to preserve 64-bit IDs safely in downstream JSON tools.

## Request model

Define a reusable request function:

```go
type RequestOptions struct {
    Method      string
    PathOrURL   string
    Query       url.Values
    Body        io.Reader
    Headers     http.Header
    Paginate    bool
    PageSize    int
    Limit       int
    DecodeInto  any
}
```

Commands should use resource-specific methods where possible:

```go
client.ListCourses(ctx, opts)
client.ListAssignments(ctx, courseID, opts)
client.SubmitAssignment(ctx, courseID, assignmentID, submission)
```

`api get` may use the lower-level request path directly.

## Pagination model

List commands should auto-paginate by default. The client must treat `Link` header URLs as opaque absolute URLs.

Pagination metadata should include:

- `paginated`
- `page_size`
- `limit`
- `request_count`
- `next_url_present`
- `total_items_returned`

## Rate-limit model

Canvas uses dynamic request cost and remaining quota headers. The client should inspect:

- `X-Request-Cost`
- `X-Rate-Limit-Remaining`
- `Retry-After`

Retry behavior:

- retry 429
- retry transient 5xx
- optionally treat 403 bodies that clearly indicate rate-limit exhaustion as retryable compatibility cases
- respect `Retry-After` when present
- use bounded exponential backoff with jitter when absent
- cap retry count

## Output model

Every command must return a typed result to the output layer. Commands should not print arbitrary JSON themselves.

Human mode:

- concise tables or lists
- no decorative noise
- warnings to stderr

JSON mode:

- stable envelope
- data under `data`
- execution metadata under `meta`
- errors under `error`

## Mutation model

Any remote write must go through safety and audit layers.

Remote writes include:

- assignment submission
- discussion post/reply
- inbox message send/reply
- grade update
- submission comment
- file upload
- course/module/page/assignment/discussion update
- publish/unpublish changes
- due date changes

Required write flow:

1. build operation plan
2. validate permissions as much as possible
3. show dry-run preview when requested
4. require confirmation unless policy allows non-interactive execution
5. execute mutation
6. capture Canvas response metadata
7. write local audit JSONL event
8. emit stable JSON envelope or human result

## File upload model

Canvas file uploads may use a multi-step upload flow. Implement a shared upload helper before implementing file submissions, bulk uploads, or submission comment attachments.

The helper should support:

- local file path validation
- size and MIME detection
- upload session creation
- upload transfer
- finalization response parsing
- checksum metadata where useful
- progress only on stderr
- no progress in JSON mode unless explicitly requested through events

## Permissions model

Canvas API access depends on user role, course role, account-level permissions, and endpoint-specific permissions. Commands should assume permission may be missing.

Permission errors are normalized:

```json
{
  "code": "CANVAS_PERMISSION_DENIED",
  "category": "permission",
  "status": 403,
  "retryable": false
}
```

## Raw API escape hatch

`canvas api get`, `api post`, `api put`, and `api delete` are available. Write operations go through safety gates.

Raw API commands should support:

- path or absolute URL under same base URL
- query params
- headers display in JSON mode
- one-page default
- `--paginate` optional
- `--raw` body output

## Core type definitions

The following Go structs define the shapes that all packages share. IDs are `string` to match Canvas string-ID mode.

### JSON envelope

```go
// Envelope is the top-level JSON output wrapper.
type Envelope struct {
    OK    bool        `json:"ok"`
    Data  interface{} `json:"data,omitempty"`
    Error *ErrorInfo  `json:"error,omitempty"`
    Meta  Meta        `json:"meta"`
}

type Meta struct {
    SchemaVersion string     `json:"schema_version"`
    Command       string     `json:"command"`
    Profile       string     `json:"profile,omitempty"`
    BaseURL       string     `json:"base_url,omitempty"`
    DurationMS    int64      `json:"duration_ms,omitempty"`
    RequestCount  int        `json:"request_count,omitempty"`
    Paginated     bool       `json:"paginated,omitempty"`
    PageSize      int        `json:"page_size,omitempty"`
    Limit         *int       `json:"limit"`
    RateLimit     *RateLimit `json:"rate_limit,omitempty"`
    Warnings      []string   `json:"warnings,omitempty"`
}

type RateLimit struct {
    RequestCost float64 `json:"request_cost"`
    Remaining   float64 `json:"remaining"`
}

type ErrorInfo struct {
    Code            string      `json:"code"`
    Message         string      `json:"message"`
    Category        string      `json:"category"`
    Retryable       bool        `json:"retryable"`
    Status          int         `json:"status,omitempty"`
    CanvasRequestID string      `json:"canvas_request_id,omitempty"`
    ResponseBody    interface{} `json:"response_body,omitempty"`
}
```

### Canvas domain types

```go
type Course struct {
    ID               string       `json:"id"`
    Name             string       `json:"name"`
    CourseCode       string       `json:"course_code"`
    WorkflowState    string       `json:"workflow_state"`
    EnrollmentTermID string       `json:"enrollment_term_id"`
    Term             *Term        `json:"term,omitempty"`
    Enrollments      []Enrollment `json:"enrollments,omitempty"`
}

type Term struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

type Assignment struct {
    ID                      string   `json:"id"`
    CourseID                string   `json:"course_id"`
    Name                    string   `json:"name"`
    DescriptionHTML         string   `json:"description_html,omitempty"`
    DueAt                   *string  `json:"due_at"`
    UnlockAt                *string  `json:"unlock_at"`
    LockAt                  *string  `json:"lock_at"`
    Published               bool     `json:"published"`
    PointsPossible          float64  `json:"points_possible"`
    SubmissionTypes          []string `json:"submission_types"`
    HasSubmittedSubmissions  bool     `json:"has_submitted_submissions"`
}

type Submission struct {
    ID             string       `json:"id"`
    UserID         string       `json:"user_id"`
    AssignmentID   string       `json:"assignment_id"`
    Score          *float64     `json:"score"`
    Grade          *string      `json:"grade"`
    SubmittedAt    *string      `json:"submitted_at"`
    WorkflowState  string       `json:"workflow_state"`
    Late           bool         `json:"late"`
    Missing        bool         `json:"missing"`
    Excused        bool         `json:"excused"`
    Attempt        *int         `json:"attempt"`
    Attachments    []Attachment `json:"attachments,omitempty"`
    User           *User        `json:"user,omitempty"`
}

type Attachment struct {
    ID          string `json:"id"`
    Filename    string `json:"filename"`
    DisplayName string `json:"display_name"`
    URL         string `json:"url"`
    Size        int64  `json:"size"`
    ContentType string `json:"content_type"`
}

type User struct {
    ID           string  `json:"id"`
    Name         string  `json:"name"`
    SortableName string  `json:"sortable_name"`
    ShortName    string  `json:"short_name"`
    Email        *string `json:"email,omitempty"`
    LoginID      string  `json:"login_id,omitempty"`
}

type Enrollment struct {
    ID                  string  `json:"id"`
    UserID              string  `json:"user_id"`
    CourseID            string  `json:"course_id"`
    Type                string  `json:"type"`
    EnrollmentState     string  `json:"enrollment_state"`
    Role                string  `json:"role"`
    Grades              *Grades `json:"grades,omitempty"`
}

type Grades struct {
    CurrentScore *float64 `json:"current_score"`
    FinalScore   *float64 `json:"final_score"`
    CurrentGrade *string  `json:"current_grade"`
    FinalGrade   *string  `json:"final_grade"`
}

type Module struct {
    ID               string `json:"id"`
    Name             string `json:"name"`
    Position         int    `json:"position"`
    Published        bool   `json:"published"`
    ItemsCount       int    `json:"items_count"`
    WorkflowState    string `json:"workflow_state"`
}

type ModuleItem struct {
    ID         string  `json:"id"`
    ModuleID   string  `json:"module_id"`
    Title      string  `json:"title"`
    Type       string  `json:"type"`
    Position   int     `json:"position"`
    ContentID  string  `json:"content_id,omitempty"`
    HTMLURL    string  `json:"html_url,omitempty"`
    URL        *string `json:"url,omitempty"`
    Published  *bool   `json:"published,omitempty"`
}

type DiscussionTopic struct {
    ID              string  `json:"id"`
    Title           string  `json:"title"`
    Message         string  `json:"message"`
    PostedAt        *string `json:"posted_at"`
    LastReplyAt     *string `json:"last_reply_at"`
    DiscussionType  string  `json:"discussion_type"`
    Published       bool    `json:"published"`
    IsAnnouncement  bool    `json:"is_announcement"`
    UserName        string  `json:"user_name,omitempty"`
}

type Page struct {
    URL          string  `json:"url"`
    Title        string  `json:"title"`
    Body         string  `json:"body,omitempty"`
    Published    bool    `json:"published"`
    CreatedAt    string  `json:"created_at"`
    UpdatedAt    string  `json:"updated_at"`
}

type Conversation struct {
    ID              string   `json:"id"`
    Subject         string   `json:"subject"`
    WorkflowState   string   `json:"workflow_state"`
    LastMessage     string   `json:"last_message"`
    LastMessageAt   string   `json:"last_message_at"`
    MessageCount    int      `json:"message_count"`
    Participants    []User   `json:"participants,omitempty"`
}

type File struct {
    ID          string `json:"id"`
    FolderID    string `json:"folder_id"`
    DisplayName string `json:"display_name"`
    Filename    string `json:"filename"`
    URL         string `json:"url"`
    Size        int64  `json:"size"`
    ContentType string `json:"content_type"`
    CreatedAt   string `json:"created_at"`
    UpdatedAt   string `json:"updated_at"`
}

type Section struct {
    ID            string `json:"id"`
    Name          string `json:"name"`
    CourseID      string `json:"course_id"`
    TotalStudents *int   `json:"total_students,omitempty"`
}

type Rubric struct {
    ID             string        `json:"id"`
    Title          string        `json:"title"`
    PointsPossible float64       `json:"points_possible"`
    Criteria       []interface{} `json:"criteria,omitempty"`
}

type AssignmentGroup struct {
    ID          string       `json:"id"`
    Name        string       `json:"name"`
    Position    int          `json:"position"`
    GroupWeight float64      `json:"group_weight"`
    Assignments []Assignment `json:"assignments,omitempty"`
}
```

### Config types

```go
type Config struct {
    CurrentProfile string             `yaml:"current_profile"`
    Profiles       map[string]Profile `yaml:"profiles"`
    Output         OutputConfig       `yaml:"output,omitempty"`
    Audit          AuditConfig        `yaml:"audit,omitempty"`
}

type Profile struct {
    BaseURL       string `yaml:"base_url"`
    Token         string `yaml:"token"`
    Timeout       string `yaml:"timeout,omitempty"`
    Retries       int    `yaml:"retries,omitempty"`
    PageSize      int    `yaml:"page_size,omitempty"`
    ReadOnly      bool   `yaml:"read_only,omitempty"`
    DefaultCourse string `yaml:"default_course,omitempty"`
}

type OutputConfig struct {
    JSONPretty bool `yaml:"json_pretty"`
    NoColor    bool `yaml:"no_color"`
}

type AuditConfig struct {
    Enabled bool   `yaml:"enabled"`
    Path    string `yaml:"path,omitempty"`
}
```

### Audit event type

```go
type AuditEvent struct {
    Time            string            `json:"time"`
    SchemaVersion   string            `json:"schema_version"`
    Command         string            `json:"command"`
    Profile         string            `json:"profile"`
    BaseURL         string            `json:"base_url"`
    Method          string            `json:"method"`
    Path            string            `json:"path"`
    Resource        map[string]string `json:"resource"`
    RequestHash     string            `json:"request_hash"`
    ResponseStatus  int               `json:"response_status"`
    CanvasRequestID string            `json:"canvas_request_id,omitempty"`
    DryRun          bool              `json:"dry_run"`
    Success         bool              `json:"success"`
}
```
