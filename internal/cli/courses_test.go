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
