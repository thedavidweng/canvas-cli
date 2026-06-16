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

func TestUsersList_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/users", 200, []map[string]any{
		{"id": "789", "name": "Alice Smith", "sortable_name": "Smith, Alice", "short_name": "Alice", "login_id": "alice@example.edu"},
		{"id": "790", "name": "Bob Jones", "sortable_name": "Jones, Bob", "short_name": "Bob", "login_id": "bob@example.edu"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newUsersListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("users list --json failed: %v", err)
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
	var users []canvas.User
	if err := json.Unmarshal(dataJSON, &users); err != nil {
		t.Fatalf("data is not []User: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
	if users[0].Name != "Alice Smith" {
		t.Errorf("expected user name 'Alice Smith', got %q", users[0].Name)
	}

	// Verify the correct API path was hit
	last := mock.LastRequest()
	if last.Path != "/api/v1/courses/1/users" {
		t.Errorf("expected request to /api/v1/courses/1/users, got %s", last.Path)
	}
}

func TestUsersList_EnrollmentTypeFilter(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/users", 200, []map[string]any{
		{"id": "789", "name": "Alice Smith", "sortable_name": "Smith, Alice", "short_name": "Alice"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newUsersListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("enrollment-type", "student")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("users list --enrollment-type student --json failed: %v", err)
	}

	// Verify the enrollment_type filter was passed in the query
	last := mock.LastRequest()
	et := last.Query.Get("enrollment_type[]")
	if et != "student" {
		t.Errorf("expected enrollment_type[]=student in query, got %q", et)
	}
}

func TestUsersList_APIError_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/users", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal server error"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newUsersListCmd()
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

func TestUsersList_APIError_Human(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/users", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal server error"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newUsersListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error in human mode")
	}
}

func TestUsersList_HumanMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/users", 200, []map[string]any{
		{"id": "789", "name": "Alice Smith", "sortable_name": "Smith, Alice", "short_name": "Alice", "login_id": "alice@example.edu"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newUsersListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("users list failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Alice Smith") {
		t.Errorf("expected 'Alice Smith' in output, got: %s", output)
	}
}

func TestUsersList_CourseRequired(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "http://localhost",
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newUsersListCmd()
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
