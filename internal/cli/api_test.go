package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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

func TestResolveData_LiteralString(t *testing.T) {
	data := `{"key":"value"}`
	got, err := resolveData(data)
	if err != nil {
		t.Fatalf("resolveData(%q) returned error: %v", data, err)
	}
	if string(got) != data {
		t.Errorf("resolveData(%q) = %q, want %q", data, got, data)
	}
}

func TestResolveData_FilePath(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "data.json")
	content := `{"hello":"world"}`
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	got, err := resolveData("@" + filePath)
	if err != nil {
		t.Fatalf("resolveData(@%s) returned error: %v", filePath, err)
	}
	if string(got) != content {
		t.Errorf("resolveData(@%s) = %q, want %q", filePath, got, content)
	}
}

func TestResolveData_NonexistentFile(t *testing.T) {
	_, err := resolveData("@/nonexistent/path/file.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

// --- api post ---

func TestApiPost_ConfirmSendsPOST(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/discussion_topics", 200, map[string]any{
		"id":    "500",
		"title": "New Topic",
	})

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newApiPostCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("data", `{"title":"New Topic"}`)
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"/api/v1/courses/1/discussion_topics"})
	if err != nil {
		t.Fatalf("api post --confirm failed: %v", err)
	}

	last := mock.LastRequest()
	if last == nil {
		t.Fatal("expected at least one request")
	}
	if last.Method != "POST" {
		t.Errorf("expected POST method, got %s", last.Method)
	}
	if last.Path != "/api/v1/courses/1/discussion_topics" {
		t.Errorf("expected path /api/v1/courses/1/discussion_topics, got %s", last.Path)
	}
	if !strings.Contains(last.Body, "New Topic") {
		t.Errorf("expected title in request body, got: %s", last.Body)
	}

	output := buf.String()
	if !strings.Contains(output, "succeeded") {
		t.Errorf("expected success message in output, got: %s", output)
	}
}

func TestApiPost_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/assignments", 200, map[string]any{
		"id":   "99",
		"name": "New Assignment",
	})

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newApiPostCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("data", `{"name":"New Assignment"}`)
	_ = cmd.Flags().Set("confirm", "true")
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, []string{"/api/v1/courses/1/assignments"})
	if err != nil {
		t.Fatalf("api post --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}
}

func TestApiPost_DryRunShowsPreview(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newApiPostCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("data", `{"title":"test"}`)
	_ = cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, []string{"/api/v1/courses/1/discussion_topics"})
	if err != nil {
		t.Fatalf("api post --dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "POST") {
		t.Errorf("expected 'POST' in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "/api/v1/courses/1/discussion_topics") {
		t.Errorf("expected endpoint path in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "title") {
		t.Errorf("expected payload preview in dry-run output, got: %s", output)
	}
	if mock.RequestCount() != 0 {
		t.Errorf("dry-run should not make HTTP requests, got %d", mock.RequestCount())
	}
}

func TestApiPost_RequiresData(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newApiPostCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"/api/v1/courses/1/assignments"})
	if err == nil {
		t.Fatal("expected error when --data is not provided")
	}
	if !strings.Contains(err.Error(), "--data") {
		t.Errorf("expected error about --data, got: %v", err)
	}
}

func TestApiPost_DataFromFile(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/assignments", 200, map[string]any{
		"id":   "99",
		"name": "File Assignment",
	})

	tmpDir := t.TempDir()
	dataPath := filepath.Join(tmpDir, "data.json")
	os.WriteFile(dataPath, []byte(`{"name":"File Assignment"}`), 0644)

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newApiPostCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("data", "@"+dataPath)
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"/api/v1/courses/1/assignments"})
	if err != nil {
		t.Fatalf("api post --data @file failed: %v", err)
	}

	last := mock.LastRequest()
	if !strings.Contains(last.Body, "File Assignment") {
		t.Errorf("expected file content in request body, got: %s", last.Body)
	}
}

// --- api put ---

func TestApiPut_ConfirmSendsPUT(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("PUT", "/api/v1/courses/1/assignments/100", 200, map[string]any{
		"id":   "100",
		"name": "Updated Assignment",
	})

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newApiPutCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("data", `{"name":"Updated Assignment"}`)
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"/api/v1/courses/1/assignments/100"})
	if err != nil {
		t.Fatalf("api put --confirm failed: %v", err)
	}

	last := mock.LastRequest()
	if last.Method != "PUT" {
		t.Errorf("expected PUT method, got %s", last.Method)
	}
	if last.Path != "/api/v1/courses/1/assignments/100" {
		t.Errorf("expected path /api/v1/courses/1/assignments/100, got %s", last.Path)
	}

	output := buf.String()
	if !strings.Contains(output, "succeeded") {
		t.Errorf("expected success message in output, got: %s", output)
	}
}

func TestApiPut_DryRunShowsPreview(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newApiPutCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("data", `{"name":"test"}`)
	_ = cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, []string{"/api/v1/courses/1/assignments/100"})
	if err != nil {
		t.Fatalf("api put --dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "PUT") {
		t.Errorf("expected 'PUT' in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "/api/v1/courses/1/assignments/100") {
		t.Errorf("expected endpoint path in dry-run output, got: %s", output)
	}
	if mock.RequestCount() != 0 {
		t.Errorf("dry-run should not make HTTP requests, got %d", mock.RequestCount())
	}
}

func TestApiPut_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("PUT", "/api/v1/courses/1/assignments/100", 200, map[string]any{
		"id":   "100",
		"name": "Updated",
	})

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newApiPutCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("data", `{"name":"Updated"}`)
	_ = cmd.Flags().Set("confirm", "true")
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, []string{"/api/v1/courses/1/assignments/100"})
	if err != nil {
		t.Fatalf("api put --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}
}

// --- api delete ---

func TestApiDelete_ConfirmSendsDELETE(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("DELETE", "/api/v1/courses/1/assignments/100", 200, map[string]any{
		"id": "100",
	})

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newApiDeleteCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"/api/v1/courses/1/assignments/100"})
	if err != nil {
		t.Fatalf("api delete --confirm failed: %v", err)
	}

	last := mock.LastRequest()
	if last.Method != "DELETE" {
		t.Errorf("expected DELETE method, got %s", last.Method)
	}
	if last.Path != "/api/v1/courses/1/assignments/100" {
		t.Errorf("expected path /api/v1/courses/1/assignments/100, got %s", last.Path)
	}

	output := buf.String()
	if !strings.Contains(output, "succeeded") {
		t.Errorf("expected success message in output, got: %s", output)
	}
}

func TestApiDelete_DryRunShowsPreview(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newApiDeleteCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, []string{"/api/v1/courses/1/assignments/100"})
	if err != nil {
		t.Fatalf("api delete --dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "DELETE") {
		t.Errorf("expected 'DELETE' in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "/api/v1/courses/1/assignments/100") {
		t.Errorf("expected endpoint path in dry-run output, got: %s", output)
	}
	if mock.RequestCount() != 0 {
		t.Errorf("dry-run should not make HTTP requests, got %d", mock.RequestCount())
	}
}

func TestApiDelete_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("DELETE", "/api/v1/courses/1/assignments/100", 200, map[string]any{
		"id": "100",
	})

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newApiDeleteCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("confirm", "true")
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, []string{"/api/v1/courses/1/assignments/100"})
	if err != nil {
		t.Fatalf("api delete --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}
}

// --- handlePaginatedRequest ---

func TestHandlePaginatedRequest_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

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
		t.Fatalf("api get --paginate --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}
	if !env.Meta.Paginated {
		t.Error("expected meta.paginated=true")
	}
	if env.Meta.RequestCount == 0 {
		t.Error("expected meta.request_count > 0")
	}
}

func TestHandlePaginatedRequest_HumanMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

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

	err := cmd.RunE(cmd, []string{"/api/v1/courses"})
	if err != nil {
		t.Fatalf("api get --paginate failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Course 1") {
		t.Errorf("expected 'Course 1' in output, got: %s", output)
	}
	if !strings.Contains(output, "Course 2") {
		t.Errorf("expected 'Course 2' in output, got: %s", output)
	}
}
