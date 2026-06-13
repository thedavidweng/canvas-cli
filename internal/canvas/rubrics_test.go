package canvas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestListRubrics(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Rubric{
			{
				ID:             "1",
				Title:          "Essay Rubric",
				PointsPossible: 100,
				Criteria: []any{
					map[string]any{"id": "c1", "description": "Thesis", "points": float64(25)},
				},
			},
			{
				ID:             "2",
				Title:          "Presentation Rubric",
				PointsPossible: 50,
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	rubrics, err := ListRubrics(context.Background(), c, "42")
	if err != nil {
		t.Fatalf("ListRubrics() error: %v", err)
	}

	if gotPath != "/api/v1/courses/42/rubrics" {
		t.Errorf("path = %q, want %q", gotPath, "/api/v1/courses/42/rubrics")
	}

	parsed, _ := url.ParseQuery(gotQuery)
	if parsed.Get("per_page") != "100" {
		t.Errorf("per_page = %q, want %q", parsed.Get("per_page"), "100")
	}

	if len(rubrics) != 2 {
		t.Fatalf("len(rubrics) = %d, want 2", len(rubrics))
	}
	if rubrics[0].ID != "1" {
		t.Errorf("rubrics[0].ID = %q, want %q", rubrics[0].ID, "1")
	}
	if rubrics[0].Title != "Essay Rubric" {
		t.Errorf("rubrics[0].Title = %q, want %q", rubrics[0].Title, "Essay Rubric")
	}
	if rubrics[0].PointsPossible != 100 {
		t.Errorf("rubrics[0].PointsPossible = %f, want 100", rubrics[0].PointsPossible)
	}
	if len(rubrics[0].Criteria) != 1 {
		t.Errorf("len(rubrics[0].Criteria) = %d, want 1", len(rubrics[0].Criteria))
	}
	if rubrics[1].ID != "2" {
		t.Errorf("rubrics[1].ID = %q, want %q", rubrics[1].ID, "2")
	}
	if rubrics[1].Title != "Presentation Rubric" {
		t.Errorf("rubrics[1].Title = %q, want %q", rubrics[1].Title, "Presentation Rubric")
	}
}

func TestListRubricsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"unauthorized"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	_, err := ListRubrics(context.Background(), c, "42")
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
}
