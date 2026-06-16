package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
	"github.com/thedavidweng/canvas-cli/internal/testutil"
)

func TestMeCmd_Exists(t *testing.T) {
	cmd := NewMeCmd()
	if cmd.Use != "me" {
		t.Errorf("expected Use 'me', got %q", cmd.Use)
	}
}

func TestMeCmd_HasGetSubcommand(t *testing.T) {
	cmd := NewMeCmd()
	found := false
	for _, sub := range cmd.Commands() {
		if sub.Name() == "get" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'get' subcommand")
	}
}

func TestMeGet_CallsAPI(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("me get failed: %v", err)
	}

	// Verify the mock received a request to /api/v1/users/self
	if mock.RequestCount() == 0 {
		t.Fatal("expected at least one request")
	}

	last := mock.LastRequest()
	if last.Path != "/api/v1/users/self" {
		t.Errorf("expected request to /api/v1/users/self, got %s", last.Path)
	}

	output := buf.String()
	if !strings.Contains(output, "Test User") {
		t.Errorf("expected user name in output, got: %s", output)
	}
}

func TestMeGet_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("me get failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}
	if env.Data == nil {
		t.Fatal("expected data in envelope")
	}
}

func TestMeGet_ShowsUserInfo(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("me get failed: %v", err)
	}

	output := buf.String()
	// Should show user info
	if !strings.Contains(output, "Test User") {
		t.Errorf("expected user name, got: %s", output)
	}
	if !strings.Contains(output, "1") {
		t.Errorf("expected user ID, got: %s", output)
	}
}

func TestMeGet_APIError(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	// Override the /api/v1/users/self to return 401
	mock.On("GET", "/api/v1/users/self", 401, map[string]any{
		"errors": []map[string]any{{"message": "Unauthorized"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "bad-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for unauthorized response")
	}
}

// --- me activity ---

func TestMeActivity_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/users/self/activity_stream", 200, []map[string]any{
		{"id": "1", "title": "New Assignment", "type": "Assignment", "message": "posted", "created_at": "2026-01-01T00:00:00Z"},
		{"id": "2", "title": "Grade Posted", "type": "Grade", "message": "", "created_at": "2026-01-02T00:00:00Z"},
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	var buf bytes.Buffer
	cmd := newMeActivityCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("me activity --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var items []canvas.ActivityItem
	if err := json.Unmarshal(dataJSON, &items); err != nil {
		t.Fatalf("data is not []ActivityItem: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}

	last := mock.LastRequest()
	if last.Path != "/api/v1/users/self/activity_stream" {
		t.Errorf("expected request to /api/v1/users/self/activity_stream, got %s", last.Path)
	}
}

func TestMeActivity_HumanMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/users/self/activity_stream", 200, []map[string]any{
		{"id": "1", "title": "New Assignment", "type": "Assignment", "message": "posted by teacher", "created_at": "2026-01-01T00:00:00Z"},
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	var buf bytes.Buffer
	cmd := newMeActivityCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("me activity failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "New Assignment") {
		t.Errorf("expected 'New Assignment' in output, got: %s", output)
	}
	if !strings.Contains(output, "Assignment") {
		t.Errorf("expected type 'Assignment' in output, got: %s", output)
	}
	if !strings.Contains(output, "posted by teacher") {
		t.Errorf("expected message in output, got: %s", output)
	}
}

// --- me todo ---

func TestMeTodo_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	dueDate := "2026-07-01T23:59:00Z"
	mock.On("GET", "/api/v1/users/self/todo", 200, []map[string]any{
		{"id": "10", "title": "Submit Essay", "type": "Assignment", "due_date": dueDate, "workflow_state": "upcoming"},
		{"id": "11", "title": "Quiz 2", "type": "Quiz", "workflow_state": "upcoming"},
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	var buf bytes.Buffer
	cmd := newMeTodoCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("me todo --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var items []canvas.TodoItem
	if err := json.Unmarshal(dataJSON, &items); err != nil {
		t.Fatalf("data is not []TodoItem: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}

	last := mock.LastRequest()
	if last.Path != "/api/v1/users/self/todo" {
		t.Errorf("expected request to /api/v1/users/self/todo, got %s", last.Path)
	}
}

func TestMeTodo_HumanMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	dueDate := "2026-07-01T23:59:00Z"
	mock.On("GET", "/api/v1/users/self/todo", 200, []map[string]any{
		{"id": "10", "title": "Submit Essay", "type": "Assignment", "due_date": dueDate, "workflow_state": "upcoming"},
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	var buf bytes.Buffer
	cmd := newMeTodoCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("me todo failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Submit Essay") {
		t.Errorf("expected 'Submit Essay' in output, got: %s", output)
	}
	if !strings.Contains(output, "2026-07-01T23:59:00Z") {
		t.Errorf("expected due date in output, got: %s", output)
	}
	if !strings.Contains(output, "Assignment") {
		t.Errorf("expected type 'Assignment' in output, got: %s", output)
	}
}

func TestMeTodo_HumanModeNoDueDate(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/users/self/todo", 200, []map[string]any{
		{"id": "11", "title": "Quiz 2", "type": "Quiz", "workflow_state": "upcoming"},
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	var buf bytes.Buffer
	cmd := newMeTodoCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("me todo failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Quiz 2") {
		t.Errorf("expected 'Quiz 2' in output, got: %s", output)
	}
	if !strings.Contains(output, "no due date") {
		t.Errorf("expected 'no due date' in output, got: %s", output)
	}
}

// --- me upcoming ---

func TestMeUpcoming_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/users/self/upcoming_events", 200, []map[string]any{
		{"id": "1", "title": "Lecture", "start_at": "2026-07-01T10:00:00Z", "end_at": "2026-07-01T11:00:00Z", "context_code": "course_1", "type": "CalendarEvent"},
		{"id": "2", "title": "Office Hours", "start_at": "2026-07-02T14:00:00Z", "context_code": "course_1", "type": "CalendarEvent"},
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	var buf bytes.Buffer
	cmd := newMeUpcomingCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("me upcoming --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var items []canvas.UpcomingEvent
	if err := json.Unmarshal(dataJSON, &items); err != nil {
		t.Fatalf("data is not []UpcomingEvent: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}

	last := mock.LastRequest()
	if last.Path != "/api/v1/users/self/upcoming_events" {
		t.Errorf("expected request to /api/v1/users/self/upcoming_events, got %s", last.Path)
	}
}

func TestMeUpcoming_HumanMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/users/self/upcoming_events", 200, []map[string]any{
		{"id": "1", "title": "Lecture", "start_at": "2026-07-01T10:00:00Z", "end_at": "2026-07-01T11:00:00Z", "context_code": "course_1", "type": "CalendarEvent"},
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	var buf bytes.Buffer
	cmd := newMeUpcomingCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("me upcoming failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Lecture") {
		t.Errorf("expected 'Lecture' in output, got: %s", output)
	}
	if !strings.Contains(output, "2026-07-01T10:00:00Z") {
		t.Errorf("expected start time in output, got: %s", output)
	}
	if !strings.Contains(output, "2026-07-01T11:00:00Z") {
		t.Errorf("expected end time in output, got: %s", output)
	}
}

func TestMeUpcoming_HumanModeNoEndAt(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/users/self/upcoming_events", 200, []map[string]any{
		{"id": "2", "title": "Office Hours", "start_at": "2026-07-02T14:00:00Z", "context_code": "course_1", "type": "CalendarEvent"},
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	var buf bytes.Buffer
	cmd := newMeUpcomingCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("me upcoming failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Office Hours") {
		t.Errorf("expected 'Office Hours' in output, got: %s", output)
	}
	// End: should NOT appear since EndAt is empty
	if strings.Contains(output, "End:") {
		t.Errorf("should not contain 'End:' when EndAt is empty, got: %s", output)
	}
}
