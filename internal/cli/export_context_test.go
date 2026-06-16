package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	// Folders
	mock.On("GET", "/api/v1/courses/1/folders", 200, []map[string]any{
		{"id": "800", "name": "Course Files", "full_name": "course files", "parent_folder_id": nil},
	})

	// Quizzes
	mock.On("GET", "/api/v1/courses/1/quizzes", 200, []map[string]any{
		{"id": "900", "title": "Quiz 1", "points_possible": 50, "published": true, "updated_at": "2026-03-01T00:00:00Z"},
	})

	// Rubrics
	mock.On("GET", "/api/v1/courses/1/rubrics", 200, []map[string]any{
		{"id": "1000", "title": "Essay Rubric", "points_possible": 100},
	})

	// Sections
	mock.On("GET", "/api/v1/courses/1/sections", 200, []map[string]any{
		{"id": "1100", "name": "Section A", "course_id": "1", "total_students": 25},
	})

	// Calendar events
	mock.On("GET", "/api/v1/calendar_events", 200, []map[string]any{
		{"id": "1200", "title": "Midterm", "start_at": "2026-04-01T09:00:00Z", "end_at": "2026-04-01T11:00:00Z", "context_code": "course_1"},
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
	if len(result.Folders) != 1 {
		t.Errorf("expected 1 folder, got %d", len(result.Folders))
	}
	if len(result.Quizzes) != 1 {
		t.Errorf("expected 1 quiz, got %d", len(result.Quizzes))
	}
	if len(result.Rubrics) != 1 {
		t.Errorf("expected 1 rubric, got %d", len(result.Rubrics))
	}
	if len(result.Sections) != 1 {
		t.Errorf("expected 1 section, got %d", len(result.Sections))
	}
	if len(result.CalendarEvents) != 1 {
		t.Errorf("expected 1 calendar event, got %d", len(result.CalendarEvents))
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

	if len(result.ExportMeta.SectionsFailed) == 0 {
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

// --- fetchPagesFromModules ---

func TestFetchPagesFromModules_ExtractsPageURLs(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	// Mock individual page fetches
	mock.On("GET", "/api/v1/courses/1/pages/syllabus", 200, map[string]any{
		"url": "syllabus", "title": "Syllabus", "body": "<p>Full syllabus</p>",
		"published": true, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-10T00:00:00Z",
	})
	mock.On("GET", "/api/v1/courses/1/pages/schedule", 200, map[string]any{
		"url": "schedule", "title": "Schedule", "body": "<p>Weekly schedule</p>",
		"published": true, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-06-01T00:00:00Z",
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{
		Modules: []any{
			map[string]any{
				"id":   "10",
				"name": "Week 1",
				"items": []any{
					map[string]any{
						"type":     "Page",
						"page_url": "syllabus",
						"title":    "Syllabus",
					},
					map[string]any{
						"type":     "Page",
						"page_url": "schedule",
						"title":    "Schedule",
					},
				},
			},
		},
	}

	reqCount, err := fetchPagesFromModules(context.Background(), client, "1", time.Time{}, result, 0)
	if err != nil {
		t.Fatalf("fetchPagesFromModules failed: %v", err)
	}
	if reqCount != 2 {
		t.Errorf("expected 2 requests, got %d", reqCount)
	}
	if result.Pages == nil {
		t.Fatal("expected pages to be populated")
	}
	if len(result.Pages) != 2 {
		t.Errorf("expected 2 pages, got %d", len(result.Pages))
	}
}

func TestFetchPagesFromModules_SkipsNonPageItems(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/pages/syllabus", 200, map[string]any{
		"url": "syllabus", "title": "Syllabus", "body": "<p>Full syllabus</p>",
		"published": true, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-10T00:00:00Z",
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{
		Modules: []any{
			map[string]any{
				"id":   "10",
				"name": "Week 1",
				"items": []any{
					map[string]any{
						"type":     "Page",
						"page_url": "syllabus",
						"title":    "Syllabus",
					},
					map[string]any{
						"type":  "Assignment",
						"title": "Essay 1",
					},
					map[string]any{
						"type":  "Quiz",
						"title": "Quiz 1",
					},
				},
			},
		},
	}

	reqCount, err := fetchPagesFromModules(context.Background(), client, "1", time.Time{}, result, 0)
	if err != nil {
		t.Fatalf("fetchPagesFromModules failed: %v", err)
	}
	// Only the Page item should be fetched
	if reqCount != 1 {
		t.Errorf("expected 1 request (only Page items), got %d", reqCount)
	}
	if len(result.Pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(result.Pages))
	}
}

func TestFetchPagesFromModules_DeduplicatesPageURLs(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/pages/syllabus", 200, map[string]any{
		"url": "syllabus", "title": "Syllabus", "body": "<p>Full syllabus</p>",
		"published": true, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-10T00:00:00Z",
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{
		Modules: []any{
			map[string]any{
				"id":   "10",
				"name": "Week 1",
				"items": []any{
					map[string]any{
						"type":     "Page",
						"page_url": "syllabus",
						"title":    "Syllabus",
					},
				},
			},
			map[string]any{
				"id":   "11",
				"name": "Week 2",
				"items": []any{
					map[string]any{
						"type":     "Page",
						"page_url": "syllabus", // duplicate
						"title":    "Syllabus Again",
					},
				},
			},
		},
	}

	reqCount, err := fetchPagesFromModules(context.Background(), client, "1", time.Time{}, result, 0)
	if err != nil {
		t.Fatalf("fetchPagesFromModules failed: %v", err)
	}
	// Only 1 request because the second URL is a duplicate
	if reqCount != 1 {
		t.Errorf("expected 1 request (duplicate skipped), got %d", reqCount)
	}
}

func TestFetchPagesFromModules_FetchErrorIncludesStub(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	// Page fetch returns 403
	mock.On("GET", "/api/v1/courses/1/pages/forbidden", 403, map[string]any{
		"errors": []map[string]any{{"message": "forbidden"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{
		Modules: []any{
			map[string]any{
				"id":   "10",
				"name": "Week 1",
				"items": []any{
					map[string]any{
						"type":     "Page",
						"page_url": "forbidden",
						"title":    "Forbidden Page",
					},
				},
			},
		},
	}

	reqCount, err := fetchPagesFromModules(context.Background(), client, "1", time.Time{}, result, 0)
	if err != nil {
		t.Fatalf("fetchPagesFromModules should not return error on fetch failure: %v", err)
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
	// Should include the stub (the original module item)
	if result.Pages == nil {
		t.Fatal("expected pages to include stub on error")
	}
	if len(result.Pages) != 1 {
		t.Errorf("expected 1 page (stub), got %d", len(result.Pages))
	}
}

func TestFetchPagesFromModules_EmptyModules(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{
		Modules: []any{},
	}

	reqCount, err := fetchPagesFromModules(context.Background(), client, "1", time.Time{}, result, 0)
	if err != nil {
		t.Fatalf("fetchPagesFromModules failed: %v", err)
	}
	if reqCount != 0 {
		t.Errorf("expected 0 requests, got %d", reqCount)
	}
	if result.Pages != nil {
		t.Errorf("expected pages to be nil for empty modules, got %v", result.Pages)
	}
}

func TestFetchPagesFromModules_SkipsEmptyPageURL(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{
		Modules: []any{
			map[string]any{
				"id":   "10",
				"name": "Week 1",
				"items": []any{
					map[string]any{
						"type":     "Page",
						"page_url": "", // empty page_url
						"title":    "Empty URL Page",
					},
				},
			},
		},
	}

	reqCount, err := fetchPagesFromModules(context.Background(), client, "1", time.Time{}, result, 0)
	if err != nil {
		t.Fatalf("fetchPagesFromModules failed: %v", err)
	}
	if reqCount != 0 {
		t.Errorf("expected 0 requests (empty page_url skipped), got %d", reqCount)
	}
}

func TestFetchPagesFromModules_InvalidModuleType(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{
		Modules: []any{
			"not a map", // invalid type
			42,          // invalid type
			map[string]any{ // valid module but no items
				"id":   "10",
				"name": "Empty Module",
			},
			map[string]any{ // valid module but items is not an array
				"id":    "11",
				"name":  "Bad Items",
				"items": "not an array",
			},
		},
	}

	reqCount, err := fetchPagesFromModules(context.Background(), client, "1", time.Time{}, result, 0)
	if err != nil {
		t.Fatalf("fetchPagesFromModules failed: %v", err)
	}
	if reqCount != 0 {
		t.Errorf("expected 0 requests (invalid modules skipped), got %d", reqCount)
	}
}

func TestFetchPagesFromModules_InvalidItemType(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{
		Modules: []any{
			map[string]any{
				"id":   "10",
				"name": "Week 1",
				"items": []any{
					"not a map", // invalid item type
					42,          // invalid item type
					map[string]any{
						"type":     "Page",
						"page_url": "valid-page",
						"title":    "Valid Page",
					},
				},
			},
		},
	}

	mock.On("GET", "/api/v1/courses/1/pages/valid-page", 200, map[string]any{
		"url": "valid-page", "title": "Valid Page", "body": "<p>Content</p>",
		"published": true, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-10T00:00:00Z",
	})

	reqCount, err := fetchPagesFromModules(context.Background(), client, "1", time.Time{}, result, 0)
	if err != nil {
		t.Fatalf("fetchPagesFromModules failed: %v", err)
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request (invalid items skipped), got %d", reqCount)
	}
	if len(result.Pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(result.Pages))
	}
}

func TestFetchPagesFromModules_SinceFilter(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/pages/old-page", 200, map[string]any{
		"url": "old-page", "title": "Old Page", "body": "<p>Old</p>",
		"published": true, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-10T00:00:00Z",
	})
	mock.On("GET", "/api/v1/courses/1/pages/new-page", 200, map[string]any{
		"url": "new-page", "title": "New Page", "body": "<p>New</p>",
		"published": true, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-06-01T00:00:00Z",
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{
		Modules: []any{
			map[string]any{
				"id":   "10",
				"name": "Week 1",
				"items": []any{
					map[string]any{
						"type":     "Page",
						"page_url": "old-page",
						"title":    "Old Page",
					},
					map[string]any{
						"type":     "Page",
						"page_url": "new-page",
						"title":    "New Page",
					},
				},
			},
		},
	}

	since, _ := time.Parse(time.RFC3339, "2026-04-01T00:00:00Z")
	reqCount, err := fetchPagesFromModules(context.Background(), client, "1", since, result, 0)
	if err != nil {
		t.Fatalf("fetchPagesFromModules failed: %v", err)
	}
	if reqCount != 2 {
		t.Errorf("expected 2 requests (both fetched), got %d", reqCount)
	}
	// Only the new page should pass the since filter
	if len(result.Pages) != 1 {
		t.Errorf("expected 1 page after since filter, got %d", len(result.Pages))
	}
}

func TestFetchPagesFromModules_BaseReqCount(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/pages/page1", 200, map[string]any{
		"url": "page1", "title": "Page 1",
		"published": true, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-10T00:00:00Z",
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{
		Modules: []any{
			map[string]any{
				"id":   "10",
				"name": "Week 1",
				"items": []any{
					map[string]any{
						"type":     "Page",
						"page_url": "page1",
						"title":    "Page 1",
					},
				},
			},
		},
	}

	// Start with a base request count of 5
	reqCount, err := fetchPagesFromModules(context.Background(), client, "1", time.Time{}, result, 5)
	if err != nil {
		t.Fatalf("fetchPagesFromModules failed: %v", err)
	}
	// 5 base + 1 page fetch = 6
	if reqCount != 6 {
		t.Errorf("expected 6 requests (5 base + 1 page), got %d", reqCount)
	}
}

// --- filterSince edge cases ---

func TestFilterSince_EmptyItems(t *testing.T) {
	since, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	result := filterSince([]any{}, since)
	if result != nil {
		t.Errorf("expected nil for empty input, got %v", result)
	}
}

func TestFilterSince_ZeroTime(t *testing.T) {
	items := []any{
		map[string]any{"id": "1", "updated_at": "2026-01-01T00:00:00Z"},
		map[string]any{"id": "2", "updated_at": "2025-01-01T00:00:00Z"},
	}
	result := filterSince(items, time.Time{})
	if len(result) != 2 {
		t.Errorf("expected all items kept with zero time, got %d", len(result))
	}
}

func TestFilterSince_AllFiltered(t *testing.T) {
	since, _ := time.Parse(time.RFC3339, "2026-12-01T00:00:00Z")
	items := []any{
		map[string]any{"id": "1", "updated_at": "2026-01-01T00:00:00Z"},
		map[string]any{"id": "2", "updated_at": "2026-02-01T00:00:00Z"},
	}
	result := filterSince(items, since)
	if len(result) != 0 {
		t.Errorf("expected all items filtered, got %d", len(result))
	}
}

func TestFilterSince_NoneFiltered(t *testing.T) {
	since, _ := time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")
	items := []any{
		map[string]any{"id": "1", "updated_at": "2026-06-01T00:00:00Z"},
		map[string]any{"id": "2", "updated_at": "2026-07-01T00:00:00Z"},
	}
	result := filterSince(items, since)
	if len(result) != 2 {
		t.Errorf("expected no items filtered, got %d", len(result))
	}
}

func TestFilterSince_NonMapItems(t *testing.T) {
	since, _ := time.Parse(time.RFC3339, "2026-06-01T00:00:00Z")
	items := []any{
		"string item",
		42,
		true,
	}
	result := filterSince(items, since)
	if len(result) != 3 {
		t.Errorf("expected all non-map items kept, got %d", len(result))
	}
}

func TestFilterSince_MissingUpdatedAt(t *testing.T) {
	since, _ := time.Parse(time.RFC3339, "2026-06-01T00:00:00Z")
	items := []any{
		map[string]any{"id": "1", "name": "no date field"},
	}
	result := filterSince(items, since)
	if len(result) != 1 {
		t.Errorf("expected item without updated_at to be kept, got %d", len(result))
	}
}

func TestFilterSince_NonStringUpdatedAt(t *testing.T) {
	since, _ := time.Parse(time.RFC3339, "2026-06-01T00:00:00Z")
	items := []any{
		map[string]any{"id": "1", "updated_at": 12345},
		map[string]any{"id": "2", "updated_at": true},
		map[string]any{"id": "3", "updated_at": nil},
	}
	result := filterSince(items, since)
	if len(result) != 3 {
		t.Errorf("expected items with non-string updated_at to be kept, got %d", len(result))
	}
}

func TestFilterSince_UnparseableDate(t *testing.T) {
	since, _ := time.Parse(time.RFC3339, "2026-06-01T00:00:00Z")
	items := []any{
		map[string]any{"id": "1", "updated_at": "not-a-date"},
		map[string]any{"id": "2", "updated_at": "2026/01/01"},
	}
	result := filterSince(items, since)
	if len(result) != 2 {
		t.Errorf("expected items with unparseable dates to be kept, got %d", len(result))
	}
}

func TestFilterSince_ExactBoundary(t *testing.T) {
	since, _ := time.Parse(time.RFC3339, "2026-06-01T00:00:00Z")
	items := []any{
		map[string]any{"id": "1", "updated_at": "2026-06-01T00:00:00Z"}, // exactly equal
		map[string]any{"id": "2", "updated_at": "2026-05-31T23:59:59Z"}, // 1 second before
	}
	result := filterSince(items, since)
	if len(result) != 1 {
		t.Errorf("expected 1 item at boundary, got %d", len(result))
	}
}

// --- classifyError ---

func TestClassifyError_Nil(t *testing.T) {
	result := classifyError(nil)
	if result != nil {
		t.Errorf("expected nil for nil input, got %v", result)
	}
}

func TestClassifyError_401String(t *testing.T) {
	err := fmt.Errorf("api error: status 401")
	result := classifyError(err)
	if !isAuthError(result) {
		t.Errorf("expected authError, got %T: %v", result, result)
	}
}

func TestClassifyError_NonAuthError(t *testing.T) {
	err := fmt.Errorf("api error: status 500")
	result := classifyError(err)
	if isAuthError(result) {
		t.Error("expected non-auth error for status 500")
	}
	if result.Error() != "api error: status 500" {
		t.Errorf("expected same error back, got %q", result.Error())
	}
}

// --- classifyStatusCode ---

func TestClassifyStatusCode_401(t *testing.T) {
	err := classifyStatusCode(401)
	if !isAuthError(err) {
		t.Error("expected authError for status 401")
	}
}

func TestClassifyStatusCode_500(t *testing.T) {
	err := classifyStatusCode(500)
	if isAuthError(err) {
		t.Error("expected non-auth error for status 500")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("expected 'status 500' in error, got %q", err.Error())
	}
}

func TestClassifyStatusCode_403(t *testing.T) {
	err := classifyStatusCode(403)
	if isAuthError(err) {
		t.Error("expected non-auth error for status 403")
	}
	if !strings.Contains(err.Error(), "status 403") {
		t.Errorf("expected 'status 403' in error, got %q", err.Error())
	}
}

// --- isAuthError ---

func TestIsAuthError_True(t *testing.T) {
	err := &authError{msg: "unauthorized"}
	if !isAuthError(err) {
		t.Error("expected true for authError")
	}
}

func TestIsAuthError_False(t *testing.T) {
	err := fmt.Errorf("some other error")
	if isAuthError(err) {
		t.Error("expected false for non-authError")
	}
}

// --- exportExitCode ---

func TestExportExitCode_NoFailures(t *testing.T) {
	result := &ExportResult{}
	if code := exportExitCode(result); code != 0 {
		t.Errorf("expected 0, got %d", code)
	}
}

func TestExportExitCode_WithFailures(t *testing.T) {
	result := &ExportResult{}
	result.ExportMeta.SectionsFailed = []string{"files"}
	if code := exportExitCode(result); code != 8 {
		t.Errorf("expected 8, got %d", code)
	}
}

// --- fetchListRaw error paths ---

func TestFetchListRaw_StatusError(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/test", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal error"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	_, reqCount, err := fetchListRaw(context.Background(), client, "/api/v1/test", nil, 100)
	if err == nil {
		t.Fatal("expected error for 500 status, got nil")
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
}

func TestFetchListRaw_401AuthError(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/test", 401, map[string]any{
		"errors": []map[string]any{{"message": "unauthorized"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	_, _, err := fetchListRaw(context.Background(), client, "/api/v1/test", nil, 100)
	if err == nil {
		t.Fatal("expected error for 401 status, got nil")
	}
	if !isAuthError(err) {
		t.Errorf("expected authError, got %T: %v", err, err)
	}
}

func TestFetchListRaw_DecodeError(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/test", 200, "not valid json{{{")

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	_, reqCount, err := fetchListRaw(context.Background(), client, "/api/v1/test", nil, 100)
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
	if !strings.Contains(err.Error(), "decode list response") {
		t.Errorf("expected 'decode list response' in error, got %q", err.Error())
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
}

func TestFetchListRaw_NilQuerySetsDefault(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/test", 200, []map[string]any{{"id": "1"}})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	items, reqCount, err := fetchListRaw(context.Background(), client, "/api/v1/test", nil, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
	last := mock.LastRequest()
	if last.Query.Get("per_page") != "50" {
		t.Errorf("expected per_page=50, got %q", last.Query.Get("per_page"))
	}
}

func TestFetchListRaw_PageSizeZero(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/test", 200, []map[string]any{{"id": "1"}})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	_, _, err := fetchListRaw(context.Background(), client, "/api/v1/test", nil, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	last := mock.LastRequest()
	if last.Query.Get("per_page") != "" {
		t.Errorf("expected no per_page for pageSize=0, got %q", last.Query.Get("per_page"))
	}
}

func TestFetchListRaw_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mock := testutil.NewMockCanvas()
	defer mock.Close()

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	_, _, err := fetchListRaw(ctx, client, "/api/v1/test", nil, 100)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

// --- fetchSingleRaw error paths ---

func TestFetchSingleRaw_StatusError(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/test", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal error"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	_, reqCount, err := fetchSingleRaw(context.Background(), client, "/api/v1/test", nil)
	if err == nil {
		t.Fatal("expected error for 500 status, got nil")
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
}

func TestFetchSingleRaw_401AuthError(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/test", 401, map[string]any{
		"errors": []map[string]any{{"message": "unauthorized"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	_, _, err := fetchSingleRaw(context.Background(), client, "/api/v1/test", nil)
	if err == nil {
		t.Fatal("expected error for 401 status, got nil")
	}
	if !isAuthError(err) {
		t.Errorf("expected authError, got %T: %v", err, err)
	}
}

func TestFetchSingleRaw_DecodeError(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/test", 200, "not valid json{{{")

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	_, reqCount, err := fetchSingleRaw(context.Background(), client, "/api/v1/test", nil)
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
	if !strings.Contains(err.Error(), "decode response") {
		t.Errorf("expected 'decode response' in error, got %q", err.Error())
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
}

// --- ExportContext edge cases ---

func TestExportContext_InvalidSince(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()
	setupExportMock(mock)

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	_, err := ExportContext(context.Background(), client, "1", ExportContextOpts{
		Since: "not-a-valid-date",
	})
	if err == nil {
		t.Fatal("expected error for invalid --since, got nil")
	}
	if !strings.Contains(err.Error(), "invalid --since value") {
		t.Errorf("expected 'invalid --since value' in error, got %q", err.Error())
	}
}

func TestExportContext_UnknownSection(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result, err := ExportContext(context.Background(), client, "1", ExportContextOpts{
		Include: []string{"nonexistent_section"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, s := range result.ExportMeta.SectionsFailed {
		if s == "nonexistent_section" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'nonexistent_section' in sections_failed, got %v", result.ExportMeta.SectionsFailed)
	}
}

func TestExportContext_EmptySections(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result, err := ExportContext(context.Background(), client, "1", ExportContextOpts{
		Include: []string{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ExportMeta.SectionsRequested) != len(allExportSections) {
		t.Errorf("expected all sections requested, got %d", len(result.ExportMeta.SectionsRequested))
	}
}

// --- fetchCourseSection error path ---

func TestFetchCourseSection_Error(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal error"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchCourseSection(context.Background(), client, "1", result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
	if result.Course != nil {
		t.Error("expected course to be nil on error")
	}
}

func TestFetchCourseSection_401(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1", 401, map[string]any{
		"errors": []map[string]any{{"message": "unauthorized"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	_, err := fetchCourseSection(context.Background(), client, "1", result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !isAuthError(err) {
		t.Errorf("expected authError, got %T: %v", err, err)
	}
}

// --- fetchTabsSection error path ---

func TestFetchTabsSection_Error(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/tabs", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal error"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchTabsSection(context.Background(), client, "1", time.Time{}, result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
}

func TestFetchTabsSection_EmptyResult(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/tabs", 200, []map[string]any{})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchTabsSection(context.Background(), client, "1", time.Time{}, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
	if result.Tabs != nil {
		t.Errorf("expected nil tabs for empty result, got %v", result.Tabs)
	}
}

// --- fetchModulesSection error paths ---

func TestFetchModulesSection_Error(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/modules", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal error"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchModulesSection(context.Background(), client, "1", time.Time{}, result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
}

func TestFetchModulesSection_Empty(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/modules", 200, []map[string]any{})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchModulesSection(context.Background(), client, "1", time.Time{}, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
	if result.Modules != nil {
		t.Errorf("expected nil modules for empty result, got %v", result.Modules)
	}
}

func TestFetchModulesSection_WithItemsAlreadyPopulated(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/modules", 200, []map[string]any{
		{
			"id":   10.0,
			"name": "Week 1",
			"items": []any{
				map[string]any{"id": "100", "title": "Item 1"},
			},
		},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchModulesSection(context.Background(), client, "1", time.Time{}, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request (items already populated), got %d", reqCount)
	}
	if len(result.Modules) != 1 {
		t.Errorf("expected 1 module, got %d", len(result.Modules))
	}
}

func TestFetchModulesSection_FetchItemsSuccess(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/modules", 200, []map[string]any{
		{"id": 10.0, "name": "Week 1", "items_count": 3.0},
	})
	mock.On("GET", "/api/v1/courses/1/modules/10/items", 200, []map[string]any{
		{"id": "100", "title": "Item 1", "type": "Page"},
		{"id": "101", "title": "Item 2", "type": "Assignment"},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchModulesSection(context.Background(), client, "1", time.Time{}, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 2 {
		t.Errorf("expected 2 requests, got %d", reqCount)
	}
	if len(result.Modules) != 1 {
		t.Errorf("expected 1 module, got %d", len(result.Modules))
	}
	mod, ok := result.Modules[0].(map[string]any)
	if !ok {
		t.Fatal("module is not a map")
	}
	items, ok := mod["items"].([]any)
	if !ok {
		t.Fatal("module items is not a slice")
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestFetchModulesSection_FetchItemsError(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/modules", 200, []map[string]any{
		{"id": 10.0, "name": "Week 1", "items_count": 3.0},
	})
	mock.On("GET", "/api/v1/courses/1/modules/10/items", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal error"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchModulesSection(context.Background(), client, "1", time.Time{}, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 2 {
		t.Errorf("expected 2 requests, got %d", reqCount)
	}
	mod, ok := result.Modules[0].(map[string]any)
	if !ok {
		t.Fatal("module is not a map")
	}
	if mod["items"] != nil {
		t.Errorf("expected items to be nil on fetch error, got %v", mod["items"])
	}
}

func TestFetchModulesSection_StringIDSkipsItemFetch(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/modules", 200, []map[string]any{
		{"id": "10", "name": "Week 1", "items_count": 3},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchModulesSection(context.Background(), client, "1", time.Time{}, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request (string id skips fetch), got %d", reqCount)
	}
}

func TestFetchModulesSection_ZeroItemsCount(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/modules", 200, []map[string]any{
		{"id": 10.0, "name": "Week 1", "items_count": 0.0},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchModulesSection(context.Background(), client, "1", time.Time{}, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request (items_count=0 skips fetch), got %d", reqCount)
	}
}

func TestFetchModulesSection_NoItemsCount(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/modules", 200, []map[string]any{
		{"id": 10.0, "name": "Week 1"},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchModulesSection(context.Background(), client, "1", time.Time{}, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request (no items_count skips fetch), got %d", reqCount)
	}
}

// --- fetchAssignmentsSection error path ---

func TestFetchAssignmentsSection_Error(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal error"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchAssignmentsSection(context.Background(), client, "1", time.Time{}, result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
}

func TestFetchAssignmentsSection_AllFiltered(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments", 200, []map[string]any{
		{"id": "100", "name": "Old", "updated_at": "2020-01-01T00:00:00Z"},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	since, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	_, err := fetchAssignmentsSection(context.Background(), client, "1", since, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Assignments != nil {
		t.Errorf("expected nil assignments when all filtered, got %v", result.Assignments)
	}
}

// --- fetchAssignmentGroupsSection error path ---

func TestFetchAssignmentGroupsSection_Error(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignment_groups", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal error"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchAssignmentGroupsSection(context.Background(), client, "1", result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
}

func TestFetchAssignmentGroupsSection_Empty(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignment_groups", 200, []map[string]any{})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchAssignmentGroupsSection(context.Background(), client, "1", result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
	if result.AssignmentGroups != nil {
		t.Errorf("expected nil for empty list, got %v", result.AssignmentGroups)
	}
}

// --- fetchFilesSection error path ---

func TestFetchFilesSection_Error(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/files", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal error"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchFilesSection(context.Background(), client, "1", time.Time{}, result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
}

func TestFetchFilesSection_AllFiltered(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/files", 200, []map[string]any{
		{"id": "1", "updated_at": "2020-01-01T00:00:00Z"},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	since, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	_, err := fetchFilesSection(context.Background(), client, "1", since, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Files != nil {
		t.Errorf("expected nil files when all filtered, got %v", result.Files)
	}
}

// --- fetchFoldersSection error path ---

func TestFetchFoldersSection_Error(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/folders", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal error"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchFoldersSection(context.Background(), client, "1", result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
}

func TestFetchFoldersSection_Empty(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/folders", 200, []map[string]any{})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchFoldersSection(context.Background(), client, "1", result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
	if result.Folders != nil {
		t.Errorf("expected nil for empty list, got %v", result.Folders)
	}
}

// --- fetchPagesSection error paths ---

func TestFetchPagesSection_ErrorWithModules(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/pages", 500, map[string]any{
		"errors": []map[string]any{{"message": "forbidden"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{
		Modules: []any{
			map[string]any{
				"id":   "10",
				"name": "Week 1",
				"items": []any{
					map[string]any{
						"type":     "Page",
						"page_url": "test-page",
						"title":    "Test Page",
					},
				},
			},
		},
	}

	reqCount, err := fetchPagesSection(context.Background(), client, "1", time.Time{}, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount == 0 {
		t.Error("expected at least 1 request")
	}
}

func TestFetchPagesSection_ErrorWithoutModules(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/pages", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal error"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	_, err := fetchPagesSection(context.Background(), client, "1", time.Time{}, result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFetchPagesSection_PageWithoutURL(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/pages", 200, []map[string]any{
		{"title": "No URL Page", "published": true},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchPagesSection(context.Background(), client, "1", time.Time{}, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
	if len(result.Pages) != 1 {
		t.Errorf("expected 1 page (without URL), got %d", len(result.Pages))
	}
}

func TestFetchPagesSection_IndividualPageFetchError(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/pages", 200, []map[string]any{
		{"url": "broken-page", "title": "Broken Page"},
	})
	mock.On("GET", "/api/v1/courses/1/pages/broken-page", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal error"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchPagesSection(context.Background(), client, "1", time.Time{}, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 2 {
		t.Errorf("expected 2 requests, got %d", reqCount)
	}
	if len(result.Pages) != 1 {
		t.Errorf("expected 1 page (stub on error), got %d", len(result.Pages))
	}
}

func TestFetchPagesSection_EmptyList(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/pages", 200, []map[string]any{})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchPagesSection(context.Background(), client, "1", time.Time{}, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
	if result.Pages != nil {
		t.Errorf("expected nil pages for empty list, got %v", result.Pages)
	}
}

// --- fetchAnnouncementsSection error path ---

func TestFetchAnnouncementsSection_Error(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/announcements", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal error"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchAnnouncementsSection(context.Background(), client, "1", time.Time{}, result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
}

func TestFetchAnnouncementsSection_AllFiltered(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/announcements", 200, []map[string]any{
		{"id": "1", "updated_at": "2020-01-01T00:00:00Z"},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	since, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	_, err := fetchAnnouncementsSection(context.Background(), client, "1", since, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Announcements != nil {
		t.Errorf("expected nil announcements when all filtered, got %v", result.Announcements)
	}
}

// --- fetchDiscussionsSection error path ---

func TestFetchDiscussionsSection_Error(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/discussion_topics", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal error"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchDiscussionsSection(context.Background(), client, "1", time.Time{}, result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
}

func TestFetchDiscussionsSection_AllFiltered(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/discussion_topics", 200, []map[string]any{
		{"id": "1", "updated_at": "2020-01-01T00:00:00Z"},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	since, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	_, err := fetchDiscussionsSection(context.Background(), client, "1", since, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Discussions != nil {
		t.Errorf("expected nil discussions when all filtered, got %v", result.Discussions)
	}
}

// --- fetchSubmissionsSection error path ---

func TestFetchSubmissionsSection_Error(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/students/submissions", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal error"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchSubmissionsSection(context.Background(), client, "1", time.Time{}, result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
}

func TestFetchSubmissionsSection_AllFiltered(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/students/submissions", 200, []map[string]any{
		{"id": "1", "updated_at": "2020-01-01T00:00:00Z"},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	since, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	_, err := fetchSubmissionsSection(context.Background(), client, "1", since, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Submissions != nil {
		t.Errorf("expected nil submissions when all filtered, got %v", result.Submissions)
	}
}

// --- fetchGradesSection error path ---

func TestFetchGradesSection_Error(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/enrollments", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal error"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchGradesSection(context.Background(), client, "1", result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
}

func TestFetchGradesSection_Empty(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/enrollments", 200, []map[string]any{})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchGradesSection(context.Background(), client, "1", result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
	if result.Grades != nil {
		t.Errorf("expected nil grades for empty list, got %v", result.Grades)
	}
}

// --- fetchQuizzesSection error path ---

func TestFetchQuizzesSection_Error(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/quizzes", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal error"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchQuizzesSection(context.Background(), client, "1", time.Time{}, result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
}

func TestFetchQuizzesSection_AllFiltered(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/quizzes", 200, []map[string]any{
		{"id": "1", "updated_at": "2020-01-01T00:00:00Z"},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	since, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	_, err := fetchQuizzesSection(context.Background(), client, "1", since, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Quizzes != nil {
		t.Errorf("expected nil quizzes when all filtered, got %v", result.Quizzes)
	}
}

// --- fetchRubricsSection error path ---

func TestFetchRubricsSection_Error(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/rubrics", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal error"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchRubricsSection(context.Background(), client, "1", result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
}

func TestFetchRubricsSection_Empty(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/rubrics", 200, []map[string]any{})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchRubricsSection(context.Background(), client, "1", result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
	if result.Rubrics != nil {
		t.Errorf("expected nil rubrics for empty list, got %v", result.Rubrics)
	}
}

// --- fetchEnrollmentsSection error path ---

func TestFetchEnrollmentsSection_Error(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/enrollments", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal error"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchEnrollmentsSection(context.Background(), client, "1", result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
}

func TestFetchEnrollmentsSection_Empty(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/enrollments", 200, []map[string]any{})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchEnrollmentsSection(context.Background(), client, "1", result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
	if result.Enrollments != nil {
		t.Errorf("expected nil enrollments for empty list, got %v", result.Enrollments)
	}
}

// --- fetchSectionsSection error path ---

func TestFetchSectionsSection_Error(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/sections", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal error"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchSectionsSection(context.Background(), client, "1", result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
}

func TestFetchSectionsSection_Empty(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/sections", 200, []map[string]any{})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchSectionsSection(context.Background(), client, "1", result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
	if result.Sections != nil {
		t.Errorf("expected nil sections for empty list, got %v", result.Sections)
	}
}

// --- fetchCalendarEventsSection error path ---

func TestFetchCalendarEventsSection_Error(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/calendar_events", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal error"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchCalendarEventsSection(context.Background(), client, "1", time.Time{}, result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
}

func TestFetchCalendarEventsSection_AllFiltered(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/calendar_events", 200, []map[string]any{
		{"id": "1", "updated_at": "2020-01-01T00:00:00Z"},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	since, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	_, err := fetchCalendarEventsSection(context.Background(), client, "1", since, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.CalendarEvents != nil {
		t.Errorf("expected nil calendar_events when all filtered, got %v", result.CalendarEvents)
	}
}

// --- fetchSection unknown section ---

func TestFetchSection_Unknown(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	result := &ExportResult{}

	reqCount, err := fetchSection(context.Background(), client, "1", "totally_unknown", time.Time{}, result)
	if err == nil {
		t.Fatal("expected error for unknown section, got nil")
	}
	if !strings.Contains(err.Error(), "unknown section") {
		t.Errorf("expected 'unknown section' in error, got %q", err.Error())
	}
	if reqCount != 0 {
		t.Errorf("expected 0 requests for unknown section, got %d", reqCount)
	}
}

// --- newCoursesExportContextCmd error paths ---

func TestNewCoursesExportContextCmd_NoConfig(t *testing.T) {
	cmd := newCoursesExportContextCmd()
	cmd.SetContext(context.Background())
	cmd.SetOut(&bytes.Buffer{})

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for missing config, got nil")
	}
	if !strings.Contains(err.Error(), "no config loaded") {
		t.Errorf("expected 'no config loaded' in error, got %q", err.Error())
	}
}

func TestNewCoursesExportContextCmd_NoCourseFlag(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "http://localhost",
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newCoursesExportContextCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for missing --course, got nil")
	}
	if !strings.Contains(err.Error(), "--course is required") {
		t.Errorf("expected '--course is required' in error, got %q", err.Error())
	}
}

func TestNewCoursesExportContextCmd_InvalidSinceFlag(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "http://localhost",
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newCoursesExportContextCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("since", "bad-date")

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for invalid --since, got nil")
	}
	if !strings.Contains(err.Error(), "invalid --since value") {
		t.Errorf("expected 'invalid --since value' in error, got %q", err.Error())
	}
}

func TestNewCoursesExportContextCmd_AuthErrorJSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1", 401, map[string]any{
		"errors": []map[string]any{{"message": "unauthorized"}},
	})

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
	_ = cmd.Flags().Set("include", "course")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error (auth error should be written to JSON, not returned): %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if env.OK {
		t.Error("expected ok:false for auth error envelope")
	}
}

func TestNewCoursesExportContextCmd_AuthErrorNonJSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1", 401, map[string]any{
		"errors": []map[string]any{{"message": "unauthorized"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newCoursesExportContextCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("include", "course")

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for auth failure in non-JSON mode, got nil")
	}
	if !isAuthError(err) {
		t.Errorf("expected authError, got %T: %v", err, err)
	}
}

func TestNewCoursesExportContextCmd_OutFileCreationError(t *testing.T) {
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
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("include", "course")
	_ = cmd.Flags().Set("out", "/nonexistent/dir/export.json")

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for bad --out path, got nil")
	}
	if !strings.Contains(err.Error(), "create output file") {
		t.Errorf("expected 'create output file' in error, got %q", err.Error())
	}
}

func TestNewCoursesExportContextCmd_RawJSONMode(t *testing.T) {
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
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("include", "course")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("export-context raw JSON mode failed: %v", err)
	}

	var result ExportResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse raw JSON: %v", err)
	}
	if result.ExportMeta.CourseID != "1" {
		t.Errorf("expected course_id '1', got %q", result.ExportMeta.CourseID)
	}
}

func TestNewCoursesExportContextCmd_IncludeFlag(t *testing.T) {
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
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("include", "course,modules")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("export-context --include failed: %v", err)
	}

	var result ExportResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(result.ExportMeta.SectionsRequested) != 2 {
		t.Errorf("expected 2 sections requested, got %d", len(result.ExportMeta.SectionsRequested))
	}
}

func TestNewCoursesExportContextCmd_SinceFlag(t *testing.T) {
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
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("include", "assignments")
	_ = cmd.Flags().Set("since", "2026-06-01T00:00:00Z")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("export-context --since failed: %v", err)
	}

	var result ExportResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(result.Assignments) != 1 {
		t.Errorf("expected 1 assignment after --since filter, got %d", len(result.Assignments))
	}
}

// --- fetchListRaw pagination ---

func TestFetchListRaw_Pagination(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.SetPagination("/api/v1/test", [][]map[string]any{
		{{"id": "1", "name": "Item 1"}},
		{{"id": "2", "name": "Item 2"}},
		{{"id": "3", "name": "Item 3"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	items, reqCount, err := fetchListRaw(context.Background(), client, "/api/v1/test", nil, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 3 {
		t.Errorf("expected 3 requests (3 pages), got %d", reqCount)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items total, got %d", len(items))
	}
}

func TestFetchListRaw_PaginationWithQuery(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.SetPagination("/api/v1/test", [][]map[string]any{
		{{"id": "1"}},
		{{"id": "2"}},
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	query := url.Values{"filter": {"active"}}
	items, reqCount, err := fetchListRaw(context.Background(), client, "/api/v1/test", query, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 2 {
		t.Errorf("expected 2 requests, got %d", reqCount)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

// --- fetchSingleRaw network error ---

func TestFetchSingleRaw_NetworkError(t *testing.T) {
	client := canvas.NewClient("http://127.0.0.1:1", "tok", "dev", 0, 0)
	_, reqCount, err := fetchSingleRaw(context.Background(), client, "/api/v1/test", nil)
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
	// reqCount should still be 1 (the attempt was made)
	if reqCount != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
}

// --- fetchListRaw pagination edge cases ---

func TestFetchListRaw_NextURLNoQuery(t *testing.T) {
	// Custom server that returns a Link header with a next URL without query params
	var baseURL string
	var callCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			// Page 1: return data with a next link that has no query params
			nextURL := baseURL + "/api/v1/test/page2"
			w.Header().Set("Link", fmt.Sprintf(`<%s>; rel="next"`, nextURL))
			w.WriteHeader(200)
			fmt.Fprint(w, `[{"id":"1"}]`)
		} else {
			// Page 2: return data without Link header
			w.WriteHeader(200)
			fmt.Fprint(w, `[{"id":"2"}]`)
		}
	}))
	defer srv.Close()
	baseURL = srv.URL

	client := canvas.NewClient(srv.URL, "tok", "dev", 0, 0)
	items, reqCount, err := fetchListRaw(context.Background(), client, "/api/v1/test", nil, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 2 {
		t.Errorf("expected 2 requests, got %d", reqCount)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestFetchListRaw_NextURLEmpty(t *testing.T) {
	// Custom server that returns a Link header with "next" but empty URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Link", `<>; rel="next"`)
		w.WriteHeader(200)
		fmt.Fprint(w, `[{"id":"1"}]`)
	}))
	defer srv.Close()

	client := canvas.NewClient(srv.URL, "tok", "dev", 0, 0)
	items, reqCount, err := fetchListRaw(context.Background(), client, "/api/v1/test", nil, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCount != 1 {
		t.Errorf("expected 1 request (empty next URL breaks), got %d", reqCount)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
}
