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

func TestApiCmd_Exists(t *testing.T) {
	cmd := NewApiCmd()
	if cmd.Use != "api" {
		t.Errorf("expected Use 'api', got %q", cmd.Use)
	}
}

func TestApiCmd_HasGetSubcommand(t *testing.T) {
	cmd := NewApiCmd()
	found := false
	for _, sub := range cmd.Commands() {
		if sub.Name() == "get" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'get' subcommand")
	}
}

func TestApiGet_CallsPath(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newApiGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{"/api/v1/courses"})
	if err != nil {
		t.Fatalf("api get failed: %v", err)
	}

	// Verify the mock received a request
	if mock.RequestCount() == 0 {
		t.Fatal("expected at least one request")
	}

	last := mock.LastRequest()
	if last.Path != "/api/v1/courses" {
		t.Errorf("expected request to /api/v1/courses, got %s", last.Path)
	}
}

func TestApiGet_PaginateAutoPaginates(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	// Set up paginated response
	mock.SetPagination("/api/v1/courses", [][]map[string]any{
		{{"id": "1", "name": "Course 1"}},
		{{"id": "2", "name": "Course 2"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL:  mock.URL(),
		Token:    "test-token",
		Profile:  "default",
		PageSize: 100,
	}

	var buf bytes.Buffer
	cmd := newApiGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("paginate", "true")
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, []string{"/api/v1/courses"})
	if err != nil {
		t.Fatalf("api get --paginate failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}
}

func TestApiGet_RawOutputsBodyOnly(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newApiGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("raw", "true")

	err := cmd.RunE(cmd, []string{"/api/v1/courses"})
	if err != nil {
		t.Fatalf("api get --raw failed: %v", err)
	}

	output := buf.String()
	// Raw mode should not have envelope wrapper
	if strings.Contains(output, "\"ok\":") {
		t.Errorf("raw mode should not have envelope, got: %s", output)
	}
	// Should contain the actual data
	if !strings.Contains(output, "Test Course") {
		t.Errorf("expected course data, got: %s", output)
	}
}

func TestApiGet_QueryAddsParams(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newApiGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("query", "state=available")

	err := cmd.RunE(cmd, []string{"/api/v1/courses"})
	if err != nil {
		t.Fatalf("api get failed: %v", err)
	}

	// Verify query param was added
	last := mock.LastRequest()
	if last.Query.Get("state") != "available" {
		t.Errorf("expected query param state=available, got: %v", last.Query)
	}
}

func TestApiGet_IncludeHeaders(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newApiGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("include-headers", "true")

	err := cmd.RunE(cmd, []string{"/api/v1/courses"})
	if err != nil {
		t.Fatalf("api get failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}
}

func TestApiGet_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newApiGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, []string{"/api/v1/courses"})
	if err != nil {
		t.Fatalf("api get failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}
	if env.Data == nil {
		t.Fatal("expected data in envelope")
	}
}

func TestApiGet_RequiresPath(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newApiGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatal("expected error when no path provided")
	}
}
