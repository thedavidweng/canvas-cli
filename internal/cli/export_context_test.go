package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
	"github.com/thedavidweng/canvas-cli/internal/testutil"
)

// setupExportMock registers all endpoints needed for a full export-context run.
func setupExportMock(mock *testutil.MockCanvas) {
	// Course with term
	mock.On("GET", "/api/v1/courses/1", 200, map[string]any{
		"id":                 "1",
		"name":               "Test Course",
		"course_code":        "TC101",
		"workflow_state":     "available",
		"enrollment_term_id": "1",
		"term": map[string]any{
			"id":   "1",
			"name": "Fall 2026",
		},
	})

	// Tabs
	mock.On("GET", "/api/v1/courses/1/tabs", 200, []map[string]any{
		{"id": "home", "label": "Home", "type": "internal", "html_url": "/courses/1", "full_url": "https://canvas.example.com/courses/1", "position": 1, "visibility": "public"},
		{"id": "modules", "label": "Modules", "type": "internal", "html_url": "/courses/1/modules", "full_url": "https://canvas.example.com/courses/1/modules", "position": 5, "visibility": "public"},
	})

	// Modules
	mock.On("GET", "/api/v1/courses/1/modules", 200, []map[string]any{
		{"id": "10", "name": "Week 1", "position": 1, "published": true, "items_count": 3, "workflow_state": "active", "updated_at": "2026-03-01T00:00:00Z"},
		{"id": "11", "name": "Week 2", "position": 2, "published": true, "items_count": 2, "workflow_state": "active", "updated_at": "2026-05-01T00:00:00Z"},
	})

	// Assignments
	mock.On("GET", "/api/v1/courses/1/assignments", 200, []map[string]any{
		{"id": "100", "name": "Essay 1", "due_at": "2026-03-15T23:59:00Z", "points_possible": 100, "published": true, "updated_at": "2026-02-01T00:00:00Z"},
		{"id": "101", "name": "Essay 2", "due_at": "2026-04-15T23:59:00Z", "points_possible": 100, "published": true, "updated_at": "2026-06-01T00:00:00Z"},
	})

	// Assignment groups
	mock.On("GET", "/api/v1/courses/1/assignment_groups", 200, []map[string]any{
		{"id": "200", "name": "Essays", "position": 1, "group_weight": 50},
	})

	// Files
	mock.On("GET", "/api/v1/courses/1/files", 200, []map[string]any{
		{"id": "300", "display_name": "syllabus.pdf", "filename": "syllabus.pdf", "content_type": "application/pdf", "size": 1024, "updated_at": "2026-01-15T00:00:00Z"},
		{"id": "301", "display_name": "slides.pptx", "filename": "slides.pptx", "content_type": "application/vnd.openxmlformats", "size": 2048, "updated_at": "2026-07-01T00:00:00Z"},
	})

	// Pages list
	mock.On("GET", "/api/v1/courses/1/pages", 200, []map[string]any{
		{"url": "syllabus", "title": "Syllabus", "published": true, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-10T00:00:00Z"},
		{"url": "schedule", "title": "Schedule", "published": true, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-06-01T00:00:00Z"},
	})
	// Individual pages (with body)
	mock.On("GET", "/api/v1/courses/1/pages/syllabus", 200, map[string]any{
		"url": "syllabus", "title": "Syllabus", "body": "<p>Course syllabus</p>", "published": true,
		"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-10T00:00:00Z",
	})
	mock.On("GET", "/api/v1/courses/1/pages/schedule", 200, map[string]any{
		"url": "schedule", "title": "Schedule", "body": "<p>Weekly schedule</p>", "published": true,
		"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-06-01T00:00:00Z",
	})

	// Announcements
	mock.On("GET", "/api/v1/announcements", 200, []map[string]any{
		{"id": "400", "title": "Welcome", "message": "Welcome to class", "is_announcement": true, "updated_at": "2026-01-05T00:00:00Z"},
	})

	// Discussions
	mock.On("GET", "/api/v1/courses/1/discussion_topics", 200, []map[string]any{
		{"id": "500", "title": "Introductions", "message": "Introduce yourself", "is_announcement": false, "updated_at": "2026-02-01T00:00:00Z"},
	})

	// Submissions
	mock.On("GET", "/api/v1/courses/1/students/submissions", 200, []map[string]any{
		{"id": "600", "user_id": "1", "assignment_id": "100", "workflow_state": "graded", "score": 95.0, "updated_at": "2026-03-20T00:00:00Z"},
	})

	// Grades (enrollments)
	mock.On("GET", "/api/v1/courses/1/enrollments", 200, []map[string]any{
		{"id": "700", "user_id": "1", "course_id": "1", "type": "StudentEnrollment", "enrollment_state": "active", "grades": map[string]any{"current_score": 92.5, "final_score": 92.5}},
	})
}

func TestExportContext_AllSections(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()
	setupExportMock(mock)

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	result, err := ExportContext(context.Background(), canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", 0, 0), "1", ExportContextOpts{})
	if err != nil {
		t.Fatalf("ExportContext failed: %v", err)
	}

	// Verify all sections are populated
	if result.Course == nil {
		t.Error("expected course data")
	}
	if len(result.Tabs) != 2 {
		t.Errorf("expected 2 tabs, got %d", len(result.Tabs))
	}
	if len(result.Modules) != 2 {
		t.Errorf("expected 2 modules, got %d", len(result.Modules))
	}
	if len(result.Assignments) != 2 {
		t.Errorf("expected 2 assignments, got %d", len(result.Assignments))
	}
	if len(result.AssignmentGroups) != 1 {
		t.Errorf("expected 1 assignment group, got %d", len(result.AssignmentGroups))
	}
	if len(result.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(result.Files))
	}
	if len(result.Pages) != 2 {
		t.Errorf("expected 2 pages, got %d", len(result.Pages))
	}
	if len(result.Announcements) != 1 {
		t.Errorf("expected 1 announcement, got %d", len(result.Announcements))
	}
	if len(result.Discussions) != 1 {
		t.Errorf("expected 1 discussion, got %d", len(result.Discussions))
	}
	if len(result.Submissions) != 1 {
		t.Errorf("expected 1 submission, got %d", len(result.Submissions))
	}
	if len(result.Grades) != 1 {
		t.Errorf("expected 1 grade, got %d", len(result.Grades))
	}

	// Verify _export_meta
	meta := result.ExportMeta
	if len(meta.SectionsSucceeded) != len(meta.SectionsRequested) {
		t.Errorf("expected all sections to succeed: requested=%v succeeded=%v", meta.SectionsRequested, meta.SectionsSucceeded)
	}
	if len(meta.SectionsFailed) != 0 {
		t.Errorf("expected no failed sections, got %v", meta.SectionsFailed)
	}
	if len(meta.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", meta.Warnings)
	}
	if meta.RequestCount == 0 {
		t.Error("expected request_count > 0")
	}
	if meta.DurationMS == 0 {
		t.Error("expected duration_ms > 0")
	}
}

func TestExportContext_IncludeFlag(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()
	setupExportMock(mock)

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	result, err := ExportContext(context.Background(), canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", 0, 0), "1", ExportContextOpts{
		Include: []string{"modules", "assignments"},
	})
	if err != nil {
		t.Fatalf("ExportContext failed: %v", err)
	}

	if len(result.Modules) != 2 {
		t.Errorf("expected 2 modules, got %d", len(result.Modules))
	}
	if len(result.Assignments) != 2 {
		t.Errorf("expected 2 assignments, got %d", len(result.Assignments))
	}
	// Sections not requested should be nil
	if result.Tabs != nil {
		t.Errorf("expected tabs to be nil when not requested, got %v", result.Tabs)
	}
	if result.Files != nil {
		t.Errorf("expected files to be nil when not requested, got %v", result.Files)
	}
	if result.Pages != nil {
		t.Errorf("expected pages to be nil when not requested, got %v", result.Pages)
	}

	// Verify meta tracks requested sections
	meta := result.ExportMeta
	if len(meta.SectionsRequested) != 2 {
		t.Errorf("expected 2 sections requested, got %d", len(meta.SectionsRequested))
	}
	if len(meta.SectionsSucceeded) != 2 {
		t.Errorf("expected 2 sections succeeded, got %d", len(meta.SectionsSucceeded))
	}
}

func TestExportContext_JSONFlag(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()
	setupExportMock(mock)

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newCoursesExportContextCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("export-context --json failed: %v", err)
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
	if env.Meta.Command != "courses.export-context" {
		t.Errorf("expected command 'courses.export-context', got %q", env.Meta.Command)
	}
}

func TestExportContext_OutFlag(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()
	setupExportMock(mock)

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "export.json")

	var buf bytes.Buffer
	cmd := newCoursesExportContextCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("out", outPath)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("export-context --out failed: %v", err)
	}

	// Stdout should be empty (file was written instead)
	if buf.Len() != 0 {
		t.Errorf("expected no stdout output, got %d bytes", buf.Len())
	}

	// File should exist and contain valid JSON
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var result ExportResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("output file is not valid ExportResult JSON: %v", err)
	}
	if result.ExportMeta.CourseID != "1" {
		t.Errorf("expected course_id '1', got %q", result.ExportMeta.CourseID)
	}
}

func TestExportContext_Section403(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()
	setupExportMock(mock)

	// Override files endpoint to return 403
	mock.On("GET", "/api/v1/courses/1/files", 403, map[string]any{
		"errors": []map[string]any{{"message": "forbidden"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	result, err := ExportContext(context.Background(), canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", 0, 0), "1", ExportContextOpts{})
	if err != nil {
		t.Fatalf("ExportContext should not return error on 403: %v", err)
	}

	// Files should be nil (section failed)
	if result.Files != nil {
		t.Errorf("expected files to be nil on 403, got %v", result.Files)
	}

	// files should be in failed sections
	found := false
	for _, s := range result.ExportMeta.SectionsFailed {
		if s == "files" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'files' in sections_failed, got %v", result.ExportMeta.SectionsFailed)
	}

	// Should have a warning
	if len(result.ExportMeta.Warnings) == 0 {
		t.Error("expected warning for 403 failure")
	}

	// Other sections should still succeed
	if result.Modules == nil {
		t.Error("modules should succeed even when files fails")
	}
	if result.Assignments == nil {
		t.Error("assignments should succeed even when files fails")
	}
}

func TestExportContext_Section401(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()
	setupExportMock(mock)

	// Override tabs to return 401 (auth failure)
	mock.On("GET", "/api/v1/courses/1/tabs", 401, map[string]any{
		"errors": []map[string]any{{"message": "unauthorized"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	_, err := ExportContext(context.Background(), canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", 0, 0), "1", ExportContextOpts{})
	if err == nil {
		t.Fatal("expected error on 401, got nil")
	}
	if !isAuthError(err) {
		t.Errorf("expected auth error, got: %v", err)
	}
}

func TestExportContext_NetworkError(t *testing.T) {
	// Use a non-existent server URL to simulate network errors
	cfg := &config.ResolvedConfig{
		BaseURL: "http://127.0.0.1:1", // port 1 should refuse connections
		Token:   "test-token",
		Profile: "default",
	}

	result, err := ExportContext(context.Background(), canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", 0, 0), "1", ExportContextOpts{
		Include: []string{"modules"}, // just one section to keep test fast
	})
	// Network errors should not return a fatal error; they warn
	if err != nil {
		t.Fatalf("ExportContext should not fail on network error (should warn): %v", err)
	}

	if result.ExportMeta.SectionsFailed == nil || len(result.ExportMeta.SectionsFailed) == 0 {
		t.Error("expected section to fail on network error")
	}

	if len(result.ExportMeta.Warnings) == 0 {
		t.Error("expected warning for network error")
	}
}

func TestExportContext_ExitCode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()
	setupExportMock(mock)

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	// All succeed -> exit 0
	result, err := ExportContext(context.Background(), canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", 0, 0), "1", ExportContextOpts{
		Include: []string{"modules"},
	})
	if err != nil {
		t.Fatalf("ExportContext failed: %v", err)
	}
	code := exportExitCode(result)
	if code != 0 {
		t.Errorf("expected exit code 0 when all succeed, got %d", code)
	}

	// Some failed -> exit 8
	mock.On("GET", "/api/v1/courses/1/files", 403, map[string]any{
		"errors": []map[string]any{{"message": "forbidden"}},
	})
	result2, err := ExportContext(context.Background(), canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", 0, 0), "1", ExportContextOpts{
		Include: []string{"modules", "files"},
	})
	if err != nil {
		t.Fatalf("ExportContext failed: %v", err)
	}
	code2 := exportExitCode(result2)
	if code2 != 8 {
		t.Errorf("expected exit code 8 on partial failure, got %d", code2)
	}
}

func TestExportContext_ExportMeta(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()
	setupExportMock(mock)

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	result, err := ExportContext(context.Background(), canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", 0, 0), "1", ExportContextOpts{
		Include: []string{"modules", "assignments"},
	})
	if err != nil {
		t.Fatalf("ExportContext failed: %v", err)
	}

	meta := result.ExportMeta
	if meta.GeneratedAt == "" {
		t.Error("expected generated_at to be set")
	}
	if meta.CourseID != "1" {
		t.Errorf("expected course_id '1', got %q", meta.CourseID)
	}
	if len(meta.SectionsRequested) != 2 {
		t.Errorf("expected 2 sections requested, got %d", len(meta.SectionsRequested))
	}
	if len(meta.SectionsSucceeded) != 2 {
		t.Errorf("expected 2 sections succeeded, got %d", len(meta.SectionsSucceeded))
	}
	if len(meta.SectionsFailed) != 0 {
		t.Errorf("expected 0 sections failed, got %d", len(meta.SectionsFailed))
	}
	if meta.DurationMS <= 0 {
		t.Errorf("expected duration_ms > 0, got %d", meta.DurationMS)
	}
	if meta.RequestCount <= 0 {
		t.Errorf("expected request_count > 0, got %d", meta.RequestCount)
	}
}

func TestExportContext_SinceFilter(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()
	setupExportMock(mock)

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	result, err := ExportContext(context.Background(), canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", 0, 0), "1", ExportContextOpts{
		Include: []string{"modules", "assignments", "files", "pages"},
		Since:   "2026-04-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("ExportContext failed: %v", err)
	}

	// Modules: only "Week 2" has updated_at=2026-05-01 >= 2026-04-01
	if len(result.Modules) != 1 {
		t.Errorf("expected 1 module after --since filter, got %d", len(result.Modules))
	}
	if len(result.Modules) > 0 {
		mod, ok := result.Modules[0].(map[string]any)
		if !ok {
			t.Fatal("module is not a map")
		}
		if mod["name"] != "Week 2" {
			t.Errorf("expected module 'Week 2', got %q", mod["name"])
		}
	}

	// Assignments: only "Essay 2" has updated_at=2026-06-01 >= 2026-04-01
	if len(result.Assignments) != 1 {
		t.Errorf("expected 1 assignment after --since filter, got %d", len(result.Assignments))
	}

	// Files: only "slides.pptx" has updated_at=2026-07-01 >= 2026-04-01
	if len(result.Files) != 1 {
		t.Errorf("expected 1 file after --since filter, got %d", len(result.Files))
	}

	// Pages: only "Schedule" has updated_at=2026-06-01 >= 2026-04-01
	if len(result.Pages) != 1 {
		t.Errorf("expected 1 page after --since filter, got %d", len(result.Pages))
	}
}
