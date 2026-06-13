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

func TestPagesCmd_Exists(t *testing.T) {
	cmd := NewPagesCmd()
	if cmd.Use != "pages" {
		t.Errorf("expected Use 'pages', got %q", cmd.Use)
	}
}

func TestPagesCmd_HasSubcommands(t *testing.T) {
	cmd := NewPagesCmd()
	subs := map[string]bool{}
	for _, sub := range cmd.Commands() {
		subs[sub.Name()] = true
	}
	for _, want := range []string{"list", "get"} {
		if !subs[want] {
			t.Errorf("expected '%s' subcommand", want)
		}
	}
}

func TestPagesList_ReturnsPages(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/pages", 200, []map[string]any{
		{
			"url":        "front-page",
			"title":      "Front Page",
			"published":  true,
			"created_at": "2026-01-01T00:00:00Z",
			"updated_at": "2026-01-01T00:00:00Z",
		},
		{
			"url":        "syllabus",
			"title":      "Syllabus",
			"published":  true,
			"created_at": "2026-01-02T00:00:00Z",
			"updated_at": "2026-01-02T00:00:00Z",
		},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newPagesListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("pages list failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Front Page") {
		t.Errorf("expected 'Front Page' in output, got: %s", output)
	}
	if !strings.Contains(output, "Syllabus") {
		t.Errorf("expected 'Syllabus' in output, got: %s", output)
	}

	last := mock.LastRequest()
	if last.Path != "/api/v1/courses/1/pages" {
		t.Errorf("expected request to /api/v1/courses/1/pages, got %s", last.Path)
	}
}

func TestPagesList_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/pages", 200, []map[string]any{
		{"url": "front-page", "title": "Front Page", "published": true},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newPagesListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("pages list --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var pages []canvas.Page
	if err := json.Unmarshal(dataJSON, &pages); err != nil {
		t.Fatalf("data is not []Page: %v", err)
	}
	if len(pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(pages))
	}
	if pages[0].Title != "Front Page" {
		t.Errorf("expected title 'Front Page', got %q", pages[0].Title)
	}
}

func TestPagesGet_ReturnsPageWithBody(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/pages/front-page", 200, map[string]any{
		"url":       "front-page",
		"title":     "Front Page",
		"body":      "<h1>Welcome</h1><p>This is the front page.</p>",
		"published": true,
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newPagesGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, []string{"front-page"})
	if err != nil {
		t.Fatalf("pages get failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Front Page") {
		t.Errorf("expected 'Front Page' in output, got: %s", output)
	}
	if !strings.Contains(output, "Welcome") {
		t.Errorf("expected 'Welcome' in output (body), got: %s", output)
	}

	last := mock.LastRequest()
	if last.Path != "/api/v1/courses/1/pages/front-page" {
		t.Errorf("expected request to /api/v1/courses/1/pages/front-page, got %s", last.Path)
	}
}

func TestPagesGet_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/pages/front-page", 200, map[string]any{
		"url":       "front-page",
		"title":     "Front Page",
		"body":      "<h1>Welcome</h1>",
		"published": true,
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newPagesGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, []string{"front-page"})
	if err != nil {
		t.Fatalf("pages get --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var page canvas.Page
	if err := json.Unmarshal(dataJSON, &page); err != nil {
		t.Fatalf("data is not Page: %v", err)
	}
	if page.URL != "front-page" {
		t.Errorf("expected URL 'front-page', got %q", page.URL)
	}
	if page.Body != "<h1>Welcome</h1>" {
		t.Errorf("expected body '<h1>Welcome</h1>', got %q", page.Body)
	}
}

func TestPagesUpdate_DryRunShowsPreview(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	bodyFile := filepath.Join(t.TempDir(), "body.html")
	if err := os.WriteFile(bodyFile, []byte("<h1>Updated Content</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newPagesUpdateCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("body-file", bodyFile)
	_ = cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, []string{"front-page"})
	if err != nil {
		t.Fatalf("pages update --dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "PUT") {
		t.Errorf("expected 'PUT' in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "/api/v1/courses/1/pages/front-page") {
		t.Errorf("expected endpoint path in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "Updated Content") {
		t.Errorf("expected body content in dry-run output, got: %s", output)
	}
	// Verify no actual request was made
	if mock.RequestCount() != 0 {
		t.Errorf("dry-run should not make HTTP requests, got %d", mock.RequestCount())
	}
}

func TestPagesUpdate_ConfirmSendsPUT(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("PUT", "/api/v1/courses/1/pages/front-page", 200, map[string]any{
		"url":       "front-page",
		"title":     "Front Page",
		"body":      "<h1>Updated Content</h1>",
		"published": true,
	})

	bodyFile := filepath.Join(t.TempDir(), "body.html")
	if err := os.WriteFile(bodyFile, []byte("<h1>Updated Content</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newPagesUpdateCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("body-file", bodyFile)
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"front-page"})
	if err != nil {
		t.Fatalf("pages update --confirm failed: %v", err)
	}

	last := mock.LastRequest()
	if last == nil {
		t.Fatal("expected at least one request")
	}
	if last.Method != "PUT" {
		t.Errorf("expected PUT method, got %s", last.Method)
	}
	if last.Path != "/api/v1/courses/1/pages/front-page" {
		t.Errorf("expected path /api/v1/courses/1/pages/front-page, got %s", last.Path)
	}
	if !strings.Contains(last.Body, "Updated Content") {
		t.Errorf("expected body in request, got: %s", last.Body)
	}

	output := buf.String()
	if !strings.Contains(output, "updated") {
		t.Errorf("expected success message in output, got: %s", output)
	}
}

func TestPagesUpdate_ReadOnlyReturnsExit7(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL:  mock.URL(),
		Token:    "test-token",
		Profile:  "default",
		ReadOnly: true,
	}

	bodyFile := filepath.Join(t.TempDir(), "body.html")
	if err := os.WriteFile(bodyFile, []byte("<h1>Content</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	cmd := newPagesUpdateCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("body-file", bodyFile)
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"front-page"})
	if err == nil {
		t.Fatal("expected error in read-only mode")
	}
	exitErr, ok := err.(interface{ ExitCode() int })
	if !ok {
		t.Fatalf("expected exit error with ExitCode(), got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 7 {
		t.Errorf("expected exit code 7, got %d", exitErr.ExitCode())
	}
}

func TestPagesUpdate_WritesAuditLog(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("PUT", "/api/v1/courses/1/pages/front-page", 200, map[string]any{
		"url":       "front-page",
		"title":     "Front Page",
		"body":      "<h1>Updated Content</h1>",
		"published": true,
	})

	bodyFile := filepath.Join(t.TempDir(), "body.html")
	if err := os.WriteFile(bodyFile, []byte("<h1>Updated Content</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}

	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    auditPath,
	}

	var buf bytes.Buffer
	cmd := newPagesUpdateCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("body-file", bodyFile)
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"front-page"})
	if err != nil {
		t.Fatalf("pages update --confirm failed: %v", err)
	}

	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("audit log is empty")
	}
	if !strings.Contains(string(data), "pages.update") {
		t.Errorf("expected 'pages.update' in audit log, got: %s", string(data))
	}
}
