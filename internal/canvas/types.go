// Package canvas defines domain types shared across all canvas-cli packages.
package canvas

import (
	"io"
	"net/http"
	"net/url"
)

// SchemaVersion is the current JSON output contract version.
const SchemaVersion = "2026-06-12"

// Envelope is the top-level JSON output wrapper.
type Envelope struct {
	OK    bool       `json:"ok"`
	Data  any        `json:"data,omitempty"`
	Error *ErrorInfo `json:"error,omitempty"`
	Meta  Meta       `json:"meta"`
}

// Meta carries execution metadata for every command response.
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

// RateLimit captures Canvas rate-limit response headers.
type RateLimit struct {
	RequestCost float64 `json:"request_cost"`
	Remaining   float64 `json:"remaining"`
}

// ErrorInfo describes a structured error returned in the JSON envelope.
type ErrorInfo struct {
	Code            string `json:"code"`
	Message         string `json:"message"`
	Category        string `json:"category"`
	Retryable       bool   `json:"retryable"`
	Status          int    `json:"status,omitempty"`
	CanvasRequestID string `json:"canvas_request_id,omitempty"`
	ResponseBody    any    `json:"response_body,omitempty"`
}

// Course represents a Canvas course.
type Course struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	CourseCode       string       `json:"course_code"`
	WorkflowState    string       `json:"workflow_state"`
	EnrollmentTermID string       `json:"enrollment_term_id"`
	Term             *Term        `json:"term,omitempty"`
	Enrollments      []Enrollment `json:"enrollments,omitempty"`
}

// Term represents an academic term.
type Term struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Assignment represents a Canvas assignment.
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
	SubmissionTypes         []string `json:"submission_types"`
	HasSubmittedSubmissions bool     `json:"has_submitted_submissions"`
}

// Submission represents a student submission.
type Submission struct {
	ID            string       `json:"id"`
	UserID        string       `json:"user_id"`
	AssignmentID  string       `json:"assignment_id"`
	Score         *float64     `json:"score"`
	Grade         *string      `json:"grade"`
	SubmittedAt   *string      `json:"submitted_at"`
	WorkflowState string       `json:"workflow_state"`
	Late          bool         `json:"late"`
	Missing       bool         `json:"missing"`
	Excused       bool         `json:"excused"`
	Attempt       *int         `json:"attempt"`
	Attachments   []Attachment `json:"attachments,omitempty"`
	User          *User        `json:"user,omitempty"`
}

// Attachment represents a file attached to a submission or other resource.
type Attachment struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	DisplayName string `json:"display_name"`
	URL         string `json:"url"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
}

// User represents a Canvas user.
type User struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	SortableName string  `json:"sortable_name"`
	ShortName    string  `json:"short_name"`
	Email        *string `json:"email,omitempty"`
	LoginID      string  `json:"login_id,omitempty"`
}

// Enrollment represents a course enrollment.
type Enrollment struct {
	ID              string  `json:"id"`
	UserID          string  `json:"user_id"`
	CourseID        string  `json:"course_id"`
	Type            string  `json:"type"`
	EnrollmentState string  `json:"enrollment_state"`
	Role            string  `json:"role"`
	Grades          *Grades `json:"grades,omitempty"`
	User            *User   `json:"user,omitempty"`
}

// Grades holds current and final grade information for an enrollment.
type Grades struct {
	CurrentScore *float64 `json:"current_score"`
	FinalScore   *float64 `json:"final_score"`
	CurrentGrade *string  `json:"current_grade"`
	FinalGrade   *string  `json:"final_grade"`
}

// Module represents a Canvas course module.
type Module struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Position      int    `json:"position"`
	Published     bool   `json:"published"`
	ItemsCount    int    `json:"items_count"`
	WorkflowState string `json:"workflow_state"`
}

// ModuleItem represents an item within a module.
type ModuleItem struct {
	ID        string  `json:"id"`
	ModuleID  string  `json:"module_id"`
	Title     string  `json:"title"`
	Type      string  `json:"type"`
	Position  int     `json:"position"`
	ContentID string  `json:"content_id,omitempty"`
	HTMLURL   string  `json:"html_url,omitempty"`
	URL       *string `json:"url,omitempty"`
	Published *bool   `json:"published,omitempty"`
}

// DiscussionTopic represents a Canvas discussion topic.
type DiscussionTopic struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Message        string  `json:"message"`
	PostedAt       *string `json:"posted_at"`
	LastReplyAt    *string `json:"last_reply_at"`
	DiscussionType string  `json:"discussion_type"`
	Published      bool    `json:"published"`
	IsAnnouncement bool    `json:"is_announcement"`
	UserName       string  `json:"user_name,omitempty"`
}

// DiscussionEntry represents a single reply/entry in a discussion topic.
type DiscussionEntry struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	UserName  string  `json:"user_name,omitempty"`
	Message   string  `json:"message"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
	ParentID  *string `json:"parent_id"`
}

// Page represents a Canvas wiki page.
type Page struct {
	URL       string `json:"url"`
	Title     string `json:"title"`
	Body      string `json:"body,omitempty"`
	Published bool   `json:"published"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Conversation represents a Canvas inbox conversation.
type Conversation struct {
	ID            string `json:"id"`
	Subject       string `json:"subject"`
	WorkflowState string `json:"workflow_state"`
	LastMessage   string `json:"last_message"`
	LastMessageAt string `json:"last_message_at"`
	MessageCount  int    `json:"message_count"`
	Participants  []User `json:"participants,omitempty"`
}

// File represents a Canvas file.
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

// Section represents a course section.
type Section struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	CourseID      string `json:"course_id"`
	TotalStudents *int   `json:"total_students,omitempty"`
}

// Rubric represents a Canvas rubric.
type Rubric struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	PointsPossible float64 `json:"points_possible"`
	Criteria       []any   `json:"criteria,omitempty"`
}

// Tab represents a course navigation tab.
type Tab struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Type       string `json:"type"`
	HTMLURL    string `json:"html_url"`
	FullURL    string `json:"full_url"`
	Position   int    `json:"position"`
	Visibility string `json:"visibility"`
}

// AssignmentGroup represents a group of assignments.
type AssignmentGroup struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Position    int          `json:"position"`
	GroupWeight float64      `json:"group_weight"`
	Assignments []Assignment `json:"assignments,omitempty"`
}

// ActivityItem represents an item in the user's activity stream.
type ActivityItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Message   string `json:"message,omitempty"`
	Type      string `json:"type"`
	ReadState string `json:"read_state,omitempty"`
	CreatedAt string `json:"created_at"`
	HTMLURL   string `json:"html_url,omitempty"`
}

// TodoItem represents a todo item for the user.
type TodoItem struct {
	Assignment    *Assignment `json:"assignment,omitempty"`
	ContextCode   string      `json:"context_code"`
	ID            string      `json:"id"`
	Title         string      `json:"title"`
	Type          string      `json:"type"`
	DueDate       *string     `json:"due_date"`
	WorkflowState string      `json:"workflow_state"`
}

// UpcomingEvent represents an upcoming calendar event.
type UpcomingEvent struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	StartAt     string `json:"start_at"`
	EndAt       string `json:"end_at,omitempty"`
	ContextCode string `json:"context_code"`
	Type        string `json:"type"`
	HTMLURL     string `json:"html_url,omitempty"`
}

// Config is the top-level configuration structure.
type Config struct {
	CurrentProfile string             `yaml:"current_profile"`
	Profiles       map[string]Profile `yaml:"profiles"`
	Output         OutputConfig       `yaml:"output,omitempty"`
	Audit          AuditConfig        `yaml:"audit,omitempty"`
}

// Profile holds per-profile connection and behavior settings.
type Profile struct {
	BaseURL       string `yaml:"base_url"`
	Token         string `yaml:"token"`
	Cookie        string `yaml:"cookie,omitempty"`
	CSRFToken     string `yaml:"csrf_token,omitempty"`
	Timeout       string `yaml:"timeout,omitempty"`
	Retries       int    `yaml:"retries,omitempty"`
	PageSize      int    `yaml:"page_size,omitempty"`
	ReadOnly      bool   `yaml:"read_only,omitempty"`
	DefaultCourse string `yaml:"default_course,omitempty"`
}

// OutputConfig controls output formatting.
type OutputConfig struct {
	JSONPretty bool `yaml:"json_pretty"`
	NoColor    bool `yaml:"no_color"`
}

// AuditConfig controls audit logging.
type AuditConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path,omitempty"`
}

// AuditEvent records a single mutation for the local audit log.
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

// RequestOptions describes a single API request to be executed by the client.
type RequestOptions struct {
	Method     string
	PathOrURL  string
	Query      url.Values
	Body       io.Reader
	Headers    http.Header
	Paginate   bool
	PageSize   int
	Limit      int
	DecodeInto any
}
