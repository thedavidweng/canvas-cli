package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
	"github.com/thedavidweng/canvas-cli/internal/testutil"
)

func TestRubricsList_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/rubrics", 200, []map[string]any{
		{
			"id":              "1",
			"title":           "Essay Rubric",
			"points_possible": 100,
			"criteria": []any{
				map[string]any{"id": "c1", "description": "Thesis", "points": float64(25)},
			},
		},
		{
			"id":              "2",
			"title":           "Presentation Rubric",
			"points_possible": 50,
		},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newRubricsListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("rubrics list --json failed: %v", err)
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
	var rubrics []canvas.Rubric
	if err := json.Unmarshal(dataJSON, &rubrics); err != nil {
		t.Fatalf("data is not []Rubric: %v", err)
	}
	if len(rubrics) != 2 {
		t.Errorf("expected 2 rubrics, got %d", len(rubrics))
	}
	if rubrics[0].Title != "Essay Rubric" {
		t.Errorf("expected rubric title 'Essay Rubric', got %q", rubrics[0].Title)
	}
	if rubrics[0].PointsPossible != 100 {
		t.Errorf("expected points_possible 100, got %f", rubrics[0].PointsPossible)
	}

	// Verify the correct API path was hit
	last := mock.LastRequest()
	if last.Path != "/api/v1/courses/1/rubrics" {
		t.Errorf("expected request to /api/v1/courses/1/rubrics, got %s", last.Path)
	}
}

func TestRubricsList_CourseRequired(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "http://localhost",
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newRubricsListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error when --course is missing")
	}
}
