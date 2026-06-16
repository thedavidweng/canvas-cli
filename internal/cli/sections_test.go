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

func TestSectionsList_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	total := 25
	mock.On("GET", "/api/v1/courses/1/sections", 200, []map[string]any{
		{"id": "10", "name": "Section A", "course_id": "1", "total_students": total},
		{"id": "11", "name": "Section B", "course_id": "1"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newSectionsListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("sections list --json failed: %v", err)
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

	dataJSON, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("failed to re-marshal data: %v", err)
	}
	var sections []canvas.Section
	if err := json.Unmarshal(dataJSON, &sections); err != nil {
		t.Fatalf("data is not []Section: %v", err)
	}
	if len(sections) != 2 {
		t.Errorf("expected 2 sections, got %d", len(sections))
	}
	if sections[0].Name != "Section A" {
		t.Errorf("expected section name 'Section A', got %q", sections[0].Name)
	}
	if sections[0].TotalStudents == nil || *sections[0].TotalStudents != 25 {
		t.Errorf("expected total_students 25, got %v", sections[0].TotalStudents)
	}

	// Verify the correct API path was hit
	last := mock.LastRequest()
	if last.Path != "/api/v1/courses/1/sections" {
		t.Errorf("expected request to /api/v1/courses/1/sections, got %s", last.Path)
	}
}

func TestSectionsList_APIError_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/sections", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal server error"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newSectionsListCmd()
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

func TestSectionsList_APIError_Human(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/sections", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal server error"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newSectionsListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error in human mode")
	}
}

func TestSectionsList_HumanMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	total := 30
	mock.On("GET", "/api/v1/courses/1/sections", 200, []map[string]any{
		{"id": "10", "name": "Section A", "course_id": "1", "total_students": total},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newSectionsListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("sections list failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Section A") {
		t.Errorf("expected 'Section A' in output, got: %s", output)
	}
	if !strings.Contains(output, "30") {
		t.Errorf("expected '30' (total students) in output, got: %s", output)
	}
}

func TestSectionsList_CourseRequired(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "http://localhost",
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newSectionsListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error when --course is missing")
	}
}
