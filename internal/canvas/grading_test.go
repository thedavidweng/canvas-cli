package canvas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSetGrade(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   map[string]map[string]any
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewDecoder(r.Body).Decode(&gotBody)

		score := 92.0
		grade := "A-"
		json.NewEncoder(w).Encode(Submission{
			ID:           "501",
			UserID:       "789",
			AssignmentID: "301",
			Score:        &score,
			Grade:        &grade,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	sub, err := SetGrade(context.Background(), c, "42", "301", "789", "92")
	if err != nil {
		t.Fatalf("SetGrade() error: %v", err)
	}

	if gotMethod != "PUT" {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	wantPath := "/api/v1/courses/42/assignments/301/submissions/789"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotBody["submission"]["posted_grade"] != "92" {
		t.Errorf("posted_grade = %v, want %q", gotBody["submission"]["posted_grade"], "92")
	}
	if sub.Score == nil || *sub.Score != 92.0 {
		t.Errorf("sub.Score = %v, want 92.0", sub.Score)
	}
	if sub.Grade == nil || *sub.Grade != "A-" {
		t.Errorf("sub.Grade = %v, want A-", sub.Grade)
	}
}

func TestAddComment(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   map[string]map[string]any
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewDecoder(r.Body).Decode(&gotBody)

		score := 85.0
		json.NewEncoder(w).Encode(Submission{
			ID:           "502",
			UserID:       "790",
			AssignmentID: "301",
			Score:        &score,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	sub, err := AddComment(context.Background(), c, "42", "301", "790", "Good work on this assignment!")
	if err != nil {
		t.Fatalf("AddComment() error: %v", err)
	}

	if gotMethod != "PUT" {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	wantPath := "/api/v1/courses/42/assignments/301/submissions/790"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotBody["comment"]["text_comment"] != "Good work on this assignment!" {
		t.Errorf("text_comment = %v, want %q", gotBody["comment"]["text_comment"], "Good work on this assignment!")
	}
	if sub.ID != "502" {
		t.Errorf("sub.ID = %q, want %q", sub.ID, "502")
	}
}

func TestGradeRubric(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   map[string]any
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewDecoder(r.Body).Decode(&gotBody)

		score := 95.0
		grade := "A"
		json.NewEncoder(w).Encode(Submission{
			ID:           "503",
			UserID:       "789",
			AssignmentID: "301",
			Score:        &score,
			Grade:        &grade,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	rubricAssessment := map[string]any{
		"crit1": map[string]any{
			"points":    10,
			"rating_id": "rat1",
		},
		"crit2": map[string]any{
			"points":    8,
			"rating_id": "rat2",
		},
	}

	sub, err := GradeRubric(context.Background(), c, "42", "301", "789", rubricAssessment)
	if err != nil {
		t.Fatalf("GradeRubric() error: %v", err)
	}

	if gotMethod != "PUT" {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	wantPath := "/api/v1/courses/42/assignments/301/submissions/789"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}

	// Verify the rubric_assessment body was sent correctly
	ra, ok := gotBody["rubric_assessment"].(map[string]any)
	if !ok {
		t.Fatalf("rubric_assessment type = %T, want map[string]any", gotBody["rubric_assessment"])
	}
	crit1, ok := ra["crit1"].(map[string]any)
	if !ok {
		t.Fatalf("crit1 type = %T, want map[string]any", ra["crit1"])
	}
	if crit1["points"] != 10.0 {
		t.Errorf("crit1.points = %v, want 10", crit1["points"])
	}
	if crit1["rating_id"] != "rat1" {
		t.Errorf("crit1.rating_id = %v, want %q", crit1["rating_id"], "rat1")
	}

	if sub.ID != "503" {
		t.Errorf("sub.ID = %q, want %q", sub.ID, "503")
	}
	if sub.Score == nil || *sub.Score != 95.0 {
		t.Errorf("sub.Score = %v, want 95.0", sub.Score)
	}
	if sub.Grade == nil || *sub.Grade != "A" {
		t.Errorf("sub.Grade = %v, want A", sub.Grade)
	}
}

func TestGradeRubricError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, err := GradeRubric(context.Background(), c, "42", "999", "789", map[string]any{})
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestImportGrades(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   map[string]any
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewDecoder(r.Body).Decode(&gotBody)

		score1 := 88.0
		score2 := 72.0
		json.NewEncoder(w).Encode([]Submission{
			{ID: "601", UserID: "789", AssignmentID: "301", Score: &score1, Grade: strPtr("B+")},
			{ID: "602", UserID: "790", AssignmentID: "301", Score: &score2, Grade: strPtr("C-")},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	gradeData := map[string]string{
		"789": "88",
		"790": "72",
	}

	result, err := ImportGrades(context.Background(), c, "42", "301", gradeData)
	if err != nil {
		t.Fatalf("ImportGrades() error: %v", err)
	}

	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	wantPath := "/api/v1/courses/42/assignments/301/submissions/update_grades"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}

	// Verify grade_data structure in request body
	gd, ok := gotBody["grade_data"].(map[string]any)
	if !ok {
		t.Fatalf("grade_data type = %T, want map[string]any", gotBody["grade_data"])
	}
	if gd["789"] != "88" {
		t.Errorf("grade_data[789] = %v, want %q", gd["789"], "88")
	}
	if gd["790"] != "72" {
		t.Errorf("grade_data[790] = %v, want %q", gd["790"], "72")
	}

	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
	if result[0].ID != "601" {
		t.Errorf("result[0].ID = %q, want %q", result[0].ID, "601")
	}
	if result[0].Score == nil || *result[0].Score != 88.0 {
		t.Errorf("result[0].Score = %v, want 88.0", result[0].Score)
	}
	if result[1].ID != "602" {
		t.Errorf("result[1].ID = %q, want %q", result[1].ID, "602")
	}
}
