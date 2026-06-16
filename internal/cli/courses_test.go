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

func TestCoursesList_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	// Register course list with two courses
	mock.On("GET", "/api/v1/courses", 200, []map[string]any{
		{"id": "1", "name": "Intro to CS", "course_code": "CS101", "workflow_state": "available"},
		{"id": "2", "name": "Data Structures", "course_code": "CS201", "workflow_state": "available"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newCoursesListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("courses list --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true in envelope")
	}
	if env.Data == nil {
		t.Fatal("expected data in envelope")
	}

	// Verify data is an array with 2 items
	dataJSON, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("failed to re-marshal data: %v", err)
	}
	var courses []canvas.Course
	if err := json.Unmarshal(dataJSON, &courses); err != nil {
		t.Fatalf("data is not []Course: %v", err)
	}
	if len(courses) != 2 {
		t.Errorf("expected 2 courses, got %d", len(courses))
	}
	if courses[0].Name != "Intro to CS" {
		t.Errorf("expected first course name 'Intro to CS', got %q", courses[0].Name)
	}
}

func TestCoursesList_HumanMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses", 200, []map[string]any{
		{"id": "1", "name": "Intro to CS", "course_code": "CS101", "workflow_state": "available"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newCoursesListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("courses list failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Intro to CS") {
		t.Errorf("expected course name in human output, got: %s", output)
	}
	if !strings.Contains(output, "CS101") {
		t.Errorf("expected course code in human output, got: %s", output)
	}
}

func TestCoursesGet_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/42", 200, map[string]any{
		"id": "42", "name": "Advanced Go", "course_code": "GO301", "workflow_state": "available",
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newCoursesGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, []string{"42"})
	if err != nil {
		t.Fatalf("courses get 42 --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var course canvas.Course
	if err := json.Unmarshal(dataJSON, &course); err != nil {
		t.Fatalf("data is not Course: %v", err)
	}
	if course.ID != "42" {
		t.Errorf("expected course ID '42', got %q", course.ID)
	}
	if course.Name != "Advanced Go" {
		t.Errorf("expected course name 'Advanced Go', got %q", course.Name)
	}
}

func TestCoursesTabs_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/tabs", 200, []map[string]any{
		{"id": "home", "label": "Home", "type": "internal", "html_url": "/courses/1", "full_url": "https://canvas.example.com/courses/1", "position": 1, "visibility": "public"},
		{"id": "modules", "label": "Modules", "type": "internal", "html_url": "/courses/1/modules", "full_url": "https://canvas.example.com/courses/1/modules", "position": 5, "visibility": "public"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newCoursesTabsCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("courses tabs --course 1 --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var tabs []canvas.Tab
	if err := json.Unmarshal(dataJSON, &tabs); err != nil {
		t.Fatalf("data is not []Tab: %v", err)
	}
	if len(tabs) != 2 {
		t.Errorf("expected 2 tabs, got %d", len(tabs))
	}
	if tabs[0].Label != "Home" {
		t.Errorf("expected first tab label 'Home', got %q", tabs[0].Label)
	}

	// Verify the correct API path was hit
	last := mock.LastRequest()
	if last.Path != "/api/v1/courses/1/tabs" {
		t.Errorf("expected request to /api/v1/courses/1/tabs, got %s", last.Path)
	}
}

func TestCoursesList_APIError_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal server error"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newCoursesListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	// In JSON mode, errors are written to output as a JSON envelope, not returned
	if err != nil {
		t.Fatalf("expected no error in JSON mode (written to output), got: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if env.OK {
		t.Error("expected ok:false on API error")
	}
}

func TestCoursesList_APIError_Human(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal server error"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newCoursesListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error in human mode")
	}
}

func TestCoursesList_EnrollmentStateFlag(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses", 200, []map[string]any{
		{"id": "1", "name": "Active Course", "course_code": "AC101", "workflow_state": "available"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newCoursesListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("enrollment-state", "active")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("courses list --enrollment-state active --json failed: %v", err)
	}

	last := mock.LastRequest()
	if last.Query.Get("enrollment_state") != "active" {
		t.Errorf("expected query param enrollment_state=active, got: %v", last.Query)
	}
}

func TestCoursesList_SearchFlag(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses", 200, []map[string]any{
		{"id": "1", "name": "Intro to CS", "course_code": "CS101", "workflow_state": "available"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newCoursesListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("search", "Intro")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("courses list --search Intro --json failed: %v", err)
	}

	last := mock.LastRequest()
	if last.Query.Get("search") != "Intro" {
		t.Errorf("expected query param search=Intro, got: %v", last.Query)
	}
}

func TestCoursesList_StateFlag(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses", 200, []map[string]any{
		{"id": "1", "name": "Completed Course", "course_code": "CC101", "workflow_state": "completed"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newCoursesListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("state", "completed")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("courses list --state completed --json failed: %v", err)
	}

	last := mock.LastRequest()
	if last.Query.Get("state") != "completed" {
		t.Errorf("expected query param state=completed, got: %v", last.Query)
	}
}

func TestCoursesList_IncludeFlag(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses", 200, []map[string]any{
		{"id": "1", "name": "Intro to CS", "course_code": "CS101", "workflow_state": "available"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newCoursesListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("include", "syllabus_body,term")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("courses list --include syllabus_body,term --json failed: %v", err)
	}

	last := mock.LastRequest()
	includes := last.Query["include[]"]
	if len(includes) != 2 {
		t.Fatalf("expected 2 include[] params, got %d: %v", len(includes), includes)
	}
	if includes[0] != "syllabus_body" || includes[1] != "term" {
		t.Errorf("expected include[]=[syllabus_body term], got %v", includes)
	}
}

func TestCoursesGet_HumanMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/42", 200, map[string]any{
		"id": "42", "name": "Advanced Go", "course_code": "GO301", "workflow_state": "available",
		"term": map[string]any{"id": "1", "name": "Fall 2026"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newCoursesGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{"42"})
	if err != nil {
		t.Fatalf("courses get 42 failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Advanced Go") {
		t.Errorf("expected course name in output, got: %s", output)
	}
	if !strings.Contains(output, "Fall 2026") {
		t.Errorf("expected term name in output, got: %s", output)
	}
}

func TestCoursesGet_APIError_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/999", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal server error"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newCoursesGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, []string{"999"})
	// In JSON mode for courses.go, error is returned (not written to output)
	// because courses.go uses writeError which returns err in non-JSON mode
	// and writeJSON in JSON mode. Actually let me check...
	// courses get uses writeError which in JSON mode writes to output and returns nil? No...
	// writeError returns err when not jsonMode, and writes JSON and returns the WriteJSON result when jsonMode
	// So in JSON mode, writeError returns the result of output.WriteJSON, which should be nil on success
	if err != nil {
		t.Fatalf("expected no error in JSON mode, got: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if env.OK {
		t.Error("expected ok:false on API error")
	}
}

func TestCoursesGet_APIError_Human(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/999", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal server error"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newCoursesGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{"999"})
	if err == nil {
		t.Fatal("expected error in human mode")
	}
}

func TestCoursesTabs_MissingCourse(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "http://localhost",
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newCoursesTabsCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error when --course is missing")
	}
	if !strings.Contains(err.Error(), "--course") {
		t.Errorf("expected error about --course, got: %v", err)
	}
}

func TestCoursesTabs_APIError_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/tabs", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal server error"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newCoursesTabsCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("expected no error in JSON mode, got: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if env.OK {
		t.Error("expected ok:false on API error")
	}
}

func TestCoursesTabs_HumanMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/tabs", 200, []map[string]any{
		{"id": "home", "label": "Home", "type": "internal", "html_url": "/courses/1", "full_url": "https://canvas.example.com/courses/1", "position": 1, "visibility": "public"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newCoursesTabsCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("courses tabs failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Home") {
		t.Errorf("expected 'Home' in output, got: %s", output)
	}
}

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"basic", "a,b,c", []string{"a", "b", "c"}},
		{"trimmed", " a , b ", []string{"a", "b"}},
		{"empty", "", nil},
		{"double comma", "a,,b", []string{"a", "b"}},
		{"single", "single", []string{"single"}},
		{"only spaces", "  ,  ,  ", nil},
		{"leading trailing comma", ",a,b,", []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitCSV(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("splitCSV(%q) = %v, want nil", tt.input, got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("splitCSV(%q) = %v (len %d), want %v (len %d)", tt.input, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitCSV(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}
