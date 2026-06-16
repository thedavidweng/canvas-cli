package output

import (
	"testing"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
)

func TestNewSuccess_CreatesEnvelopeWithOKTrue(t *testing.T) {
	data := []string{"a", "b"}
	env := NewSuccess(data, "courses.list")

	if !env.OK {
		t.Fatal("expected ok=true")
	}
	if env.Data == nil {
		t.Fatal("expected data to be set")
	}
	if env.Error != nil {
		t.Fatalf("expected no error, got %+v", env.Error)
	}
}

func TestNewSuccess_SetsMetaFields(t *testing.T) {
	env := NewSuccess(nil, "assignments.get")

	if env.Meta.SchemaVersion != SchemaVersion {
		t.Errorf("schema_version = %q, want %q", env.Meta.SchemaVersion, SchemaVersion)
	}
	if env.Meta.Command != "assignments.get" {
		t.Errorf("command = %q, want %q", env.Meta.Command, "assignments.get")
	}
}

func TestNewSuccess_AppliesMetaOverrides(t *testing.T) {
	rl := &canvas.RateLimit{RequestCost: 1.5, Remaining: 500}
	overrides := canvas.Meta{
		Profile:      "staging",
		BaseURL:      "https://staging.instructure.com",
		DurationMS:   42,
		RequestCount: 3,
		Paginated:    true,
		PageSize:     100,
		RateLimit:    rl,
		Warnings:     []string{"partial data"},
	}

	env := NewSuccess([]int{1}, "courses.list", overrides)

	if env.Meta.Profile != "staging" {
		t.Errorf("profile = %q, want %q", env.Meta.Profile, "staging")
	}
	if env.Meta.BaseURL != "https://staging.instructure.com" {
		t.Errorf("base_url = %q, want %q", env.Meta.BaseURL, "https://staging.instructure.com")
	}
	if env.Meta.DurationMS != 42 {
		t.Errorf("duration_ms = %d, want 42", env.Meta.DurationMS)
	}
	if env.Meta.RequestCount != 3 {
		t.Errorf("request_count = %d, want 3", env.Meta.RequestCount)
	}
	if !env.Meta.Paginated {
		t.Error("expected paginated=true")
	}
	if env.Meta.PageSize != 100 {
		t.Errorf("page_size = %d, want 100", env.Meta.PageSize)
	}
	if env.Meta.RateLimit == nil {
		t.Fatal("expected rate_limit to be set")
	}
	if env.Meta.RateLimit.RequestCost != 1.5 {
		t.Errorf("rate_limit.request_cost = %f, want 1.5", env.Meta.RateLimit.RequestCost)
	}
	if env.Meta.RateLimit.Remaining != 500 {
		t.Errorf("rate_limit.remaining = %f, want 500", env.Meta.RateLimit.Remaining)
	}
	if len(env.Meta.Warnings) != 1 || env.Meta.Warnings[0] != "partial data" {
		t.Errorf("warnings = %v, want [partial data]", env.Meta.Warnings)
	}
}

func TestNewError_CreatesEnvelopeWithOKFalse(t *testing.T) {
	errInfo := canvas.ErrorInfo{
		Code:      "CANVAS_API_ERROR",
		Message:   "Canvas API request failed",
		Category:  "api",
		Retryable: false,
		Status:    400,
	}

	env := NewError(errInfo, "assignments.list")

	if env.OK {
		t.Fatal("expected ok=false")
	}
	if env.Error == nil {
		t.Fatal("expected error to be set")
	}
	if env.Error.Code != "CANVAS_API_ERROR" {
		t.Errorf("error.code = %q, want %q", env.Error.Code, "CANVAS_API_ERROR")
	}
	if env.Data != nil {
		t.Fatalf("expected no data, got %+v", env.Data)
	}
}

func TestNewError_SetsMetaFields(t *testing.T) {
	errInfo := canvas.ErrorInfo{
		Code:     "AUTH_ERROR",
		Message:  "invalid token",
		Category: "auth",
	}

	env := NewError(errInfo, "courses.list")

	if env.Meta.SchemaVersion != SchemaVersion {
		t.Errorf("schema_version = %q, want %q", env.Meta.SchemaVersion, SchemaVersion)
	}
	if env.Meta.Command != "courses.list" {
		t.Errorf("command = %q, want %q", env.Meta.Command, "courses.list")
	}
}

func TestNewError_AppliesMetaOverrides(t *testing.T) {
	errInfo := canvas.ErrorInfo{Code: "RATE_LIMIT", Message: "too many requests", Category: "api"}
	overrides := canvas.Meta{
		DurationMS:   100,
		RequestCount: 5,
		RateLimit:    &canvas.RateLimit{RequestCost: 2.0, Remaining: 0},
	}

	env := NewError(errInfo, "submissions.list", overrides)

	if env.Meta.DurationMS != 100 {
		t.Errorf("duration_ms = %d, want 100", env.Meta.DurationMS)
	}
	if env.Meta.RequestCount != 5 {
		t.Errorf("request_count = %d, want 5", env.Meta.RequestCount)
	}
	if env.Meta.RateLimit == nil || env.Meta.RateLimit.Remaining != 0 {
		t.Error("expected rate_limit with remaining=0")
	}
}

func TestNewSuccess_LimitOverride(t *testing.T) {
	limit := 25
	overrides := canvas.Meta{
		Limit: &limit,
	}

	env := NewSuccess([]int{1}, "courses.list", overrides)

	if env.Meta.Limit == nil {
		t.Fatal("expected limit to be set")
	}
	if *env.Meta.Limit != 25 {
		t.Errorf("limit = %d, want 25", *env.Meta.Limit)
	}
}

func TestNewSuccess_NoOverrides(t *testing.T) {
	env := NewSuccess("data", "test.cmd")

	if env.Meta.SchemaVersion != SchemaVersion {
		t.Errorf("schema_version = %q, want %q", env.Meta.SchemaVersion, SchemaVersion)
	}
	if env.Meta.Command != "test.cmd" {
		t.Errorf("command = %q, want %q", env.Meta.Command, "test.cmd")
	}
	if env.Meta.Limit != nil {
		t.Error("limit should be nil when not overridden")
	}
}

func TestNewError_IncludesCanvasRequestID(t *testing.T) {
	errInfo := canvas.ErrorInfo{
		Code:            "CANVAS_API_ERROR",
		Message:         "bad request",
		Category:        "api",
		CanvasRequestID: "req-abc-123",
		ResponseBody:    map[string]any{"errors": []any{}},
	}

	env := NewError(errInfo, "api.get")

	if env.Error.CanvasRequestID != "req-abc-123" {
		t.Errorf("canvas_request_id = %q, want %q", env.Error.CanvasRequestID, "req-abc-123")
	}
	if env.Error.ResponseBody == nil {
		t.Error("expected response_body to be set")
	}
}
