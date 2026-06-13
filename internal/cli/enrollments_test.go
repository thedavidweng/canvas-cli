package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
	"github.com/thedavidweng/canvas-cli/internal/testutil"
)

func TestEnrollmentsList_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	score := 92.5
	grade := "A"
	mock.On("GET", "/api/v1/courses/1/enrollments", 200, []map[string]any{
		{
			"id":               "100",
			"user_id":          "789",
			"course_id":        "1",
			"type":             "StudentEnrollment",
			"enrollment_state": "active",
			"role":             "StudentEnrollment",
			"grades": map[string]any{
				"current_score": score,
				"current_grade": grade,
			},
			"user": map[string]any{
				"id":            "789",
				"name":          "Alice Smith",
				"sortable_name": "Smith, Alice",
				"short_name":    "Alice",
			},
		},
		{
			"id":               "101",
			"user_id":          "790",
			"course_id":        "1",
			"type":             "StudentEnrollment",
			"enrollment_state": "active",
			"role":             "StudentEnrollment",
			"user": map[string]any{
				"id":            "790",
				"name":          "Bob Jones",
				"sortable_name": "Jones, Bob",
				"short_name":    "Bob",
			},
		},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newEnrollmentsListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("enrollments list --json failed: %v", err)
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
	var enrollments []canvas.Enrollment
	if err := json.Unmarshal(dataJSON, &enrollments); err != nil {
		t.Fatalf("data is not []Enrollment: %v", err)
	}
	if len(enrollments) != 2 {
		t.Errorf("expected 2 enrollments, got %d", len(enrollments))
	}
	if enrollments[0].Role != "StudentEnrollment" {
		t.Errorf("expected role 'StudentEnrollment', got %q", enrollments[0].Role)
	}
	if enrollments[0].User == nil {
		t.Fatal("expected user in first enrollment")
	}
	if enrollments[0].User.Name != "Alice Smith" {
		t.Errorf("expected user name 'Alice Smith', got %q", enrollments[0].User.Name)
	}

	// Verify the correct API path was hit
	last := mock.LastRequest()
	if last.Path != "/api/v1/courses/1/enrollments" {
		t.Errorf("expected request to /api/v1/courses/1/enrollments, got %s", last.Path)
	}
}

func TestEnrollmentsList_HumanMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	score := 92.5
	grade := "A"
	mock.On("GET", "/api/v1/courses/1/enrollments", 200, []map[string]any{
		{
			"id":               "100",
			"user_id":          "789",
			"course_id":        "1",
			"type":             "StudentEnrollment",
			"enrollment_state": "active",
			"role":             "StudentEnrollment",
			"grades": map[string]any{
				"current_score": score,
				"current_grade": grade,
			},
			"user": map[string]any{
				"id":            "789",
				"name":          "Alice Smith",
				"sortable_name": "Smith, Alice",
				"short_name":    "Alice",
			},
		},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newEnrollmentsListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("enrollments list failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Alice Smith") {
		t.Errorf("expected 'Alice Smith' in human output, got: %s", output)
	}
	if !strings.Contains(output, "StudentEnrollment") {
		t.Errorf("expected 'StudentEnrollment' in human output, got: %s", output)
	}
	if !strings.Contains(output, fmt.Sprintf("%.1f", score)) || !strings.Contains(output, grade) {
		t.Errorf("expected grade info in human output, got: %s", output)
	}
}

func TestEnrollmentsList_CourseRequired(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "http://localhost",
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newEnrollmentsListCmd()
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
