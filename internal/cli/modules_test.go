package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
	"github.com/thedavidweng/canvas-cli/internal/testutil"
)

func TestModulesList_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/modules", 200, []map[string]any{
		{"id": "10", "name": "Week 1", "position": 1, "published": true, "items_count": 3, "workflow_state": "active"},
		{"id": "11", "name": "Week 2", "position": 2, "published": true, "items_count": 5, "workflow_state": "active"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newModulesListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("modules list --course 1 --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var modules []canvas.Module
	if err := json.Unmarshal(dataJSON, &modules); err != nil {
		t.Fatalf("data is not []Module: %v", err)
	}
	if len(modules) != 2 {
		t.Errorf("expected 2 modules, got %d", len(modules))
	}
	if modules[0].Name != "Week 1" {
		t.Errorf("expected module name 'Week 1', got %q", modules[0].Name)
	}
}

func TestModulesItems_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/modules/10/items", 200, []map[string]any{
		{"id": "100", "module_id": "10", "title": "Introduction", "type": "Page", "position": 1},
		{"id": "101", "module_id": "10", "title": "Quiz 1", "type": "Quiz", "position": 2},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newModulesItemsCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("module", "10")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("modules items --course 1 --module 10 --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var items []canvas.ModuleItem
	if err := json.Unmarshal(dataJSON, &items); err != nil {
		t.Fatalf("data is not []ModuleItem: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
	if items[0].Title != "Introduction" {
		t.Errorf("expected item title 'Introduction', got %q", items[0].Title)
	}

	// Verify the correct API path was hit
	last := mock.LastRequest()
	if last.Path != "/api/v1/courses/1/modules/10/items" {
		t.Errorf("expected request to /api/v1/courses/1/modules/10/items, got %s", last.Path)
	}
}

func TestModulesGet_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/modules/10", 200, map[string]any{
		"id": "10", "name": "Week 1", "position": 1, "published": true, "items_count": 3, "workflow_state": "active",
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newModulesGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, []string{"10"})
	if err != nil {
		t.Fatalf("modules get --course 1 10 --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var mod canvas.Module
	if err := json.Unmarshal(dataJSON, &mod); err != nil {
		t.Fatalf("data is not Module: %v", err)
	}
	if mod.ID != "10" {
		t.Errorf("expected module ID '10', got %q", mod.ID)
	}
	if mod.Name != "Week 1" {
		t.Errorf("expected module name 'Week 1', got %q", mod.Name)
	}

	// Verify the correct API path was hit
	last := mock.LastRequest()
	if last.Path != "/api/v1/courses/1/modules/10" {
		t.Errorf("expected request to /api/v1/courses/1/modules/10, got %s", last.Path)
	}
}

func TestModulesPublish_DryRunShowsPreview(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newModulesPublishCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, []string{"10"})
	if err != nil {
		t.Fatalf("modules publish --dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "PUT") {
		t.Errorf("expected 'PUT' in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "/api/v1/courses/1/modules/10") {
		t.Errorf("expected endpoint path in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "published=true") {
		t.Errorf("expected published=true in dry-run output, got: %s", output)
	}
	// Verify no actual request was made
	if mock.RequestCount() != 0 {
		t.Errorf("dry-run should not make HTTP requests, got %d", mock.RequestCount())
	}
}

func TestModulesPublish_ConfirmSendsPUT(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("PUT", "/api/v1/courses/1/modules/10", 200, map[string]any{
		"id":             "10",
		"name":           "Week 1",
		"position":       1,
		"published":      true,
		"items_count":    3,
		"workflow_state": "active",
	})

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newModulesPublishCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"10"})
	if err != nil {
		t.Fatalf("modules publish --confirm failed: %v", err)
	}

	last := mock.LastRequest()
	if last == nil {
		t.Fatal("expected at least one request")
	}
	if last.Method != "PUT" {
		t.Errorf("expected PUT method, got %s", last.Method)
	}
	if last.Path != "/api/v1/courses/1/modules/10" {
		t.Errorf("expected path /api/v1/courses/1/modules/10, got %s", last.Path)
	}
	if !strings.Contains(last.Body, `"published":true`) && !strings.Contains(last.Body, `"published": true`) {
		t.Errorf("expected published=true in request body, got: %s", last.Body)
	}

	output := buf.String()
	if !strings.Contains(output, "published") {
		t.Errorf("expected success message in output, got: %s", output)
	}
}

func TestModulesUnpublish_DryRunShowsPreview(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newModulesUnpublishCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, []string{"10"})
	if err != nil {
		t.Fatalf("modules unpublish --dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "PUT") {
		t.Errorf("expected 'PUT' in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "/api/v1/courses/1/modules/10") {
		t.Errorf("expected endpoint path in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "published=false") {
		t.Errorf("expected published=false in dry-run output, got: %s", output)
	}
	// Verify no actual request was made
	if mock.RequestCount() != 0 {
		t.Errorf("dry-run should not make HTTP requests, got %d", mock.RequestCount())
	}
}

func TestModulesUnpublish_ConfirmSendsPUT(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("PUT", "/api/v1/courses/1/modules/10", 200, map[string]any{
		"id":             "10",
		"name":           "Week 1",
		"position":       1,
		"published":      false,
		"items_count":    3,
		"workflow_state": "active",
	})

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newModulesUnpublishCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"10"})
	if err != nil {
		t.Fatalf("modules unpublish --confirm failed: %v", err)
	}

	last := mock.LastRequest()
	if last == nil {
		t.Fatal("expected at least one request")
	}
	if last.Method != "PUT" {
		t.Errorf("expected PUT method, got %s", last.Method)
	}
	if last.Path != "/api/v1/courses/1/modules/10" {
		t.Errorf("expected path /api/v1/courses/1/modules/10, got %s", last.Path)
	}
	if !strings.Contains(last.Body, `"published":false`) && !strings.Contains(last.Body, `"published": false`) {
		t.Errorf("expected published=false in request body, got: %s", last.Body)
	}

	output := buf.String()
	if !strings.Contains(output, "unpublished") {
		t.Errorf("expected success message in output, got: %s", output)
	}
}

// --- modules list human mode ---

func TestModulesList_HumanMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/modules", 200, []map[string]any{
		{"id": "10", "name": "Week 1", "position": 1, "published": true, "items_count": 3, "workflow_state": "active"},
		{"id": "11", "name": "Week 2", "position": 2, "published": false, "items_count": 5, "workflow_state": "active"},
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	var buf bytes.Buffer
	cmd := newModulesListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("modules list failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Week 1") {
		t.Errorf("expected 'Week 1' in output, got: %s", output)
	}
	if !strings.Contains(output, "Week 2") {
		t.Errorf("expected 'Week 2' in output, got: %s", output)
	}
}

// --- modules get human mode ---

func TestModulesGet_HumanMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/modules/10", 200, map[string]any{
		"id": "10", "name": "Week 1", "position": 1, "published": true, "items_count": 3, "workflow_state": "active",
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	var buf bytes.Buffer
	cmd := newModulesGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, []string{"10"})
	if err != nil {
		t.Fatalf("modules get failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Week 1") {
		t.Errorf("expected 'Week 1' in output, got: %s", output)
	}
	if !strings.Contains(output, "3") {
		t.Errorf("expected items count '3' in output, got: %s", output)
	}
}

// --- modules item ---

func TestModulesItem_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/modules/10/items/100", 200, map[string]any{
		"id":         "100",
		"module_id":  "10",
		"title":      "Introduction",
		"type":       "Page",
		"position":   1,
		"content_id": "50",
		"html_url":   "/courses/1/pages/introduction",
		"published":  true,
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	var buf bytes.Buffer
	cmd := newModulesItemCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("module", "10")

	err := cmd.RunE(cmd, []string{"100"})
	if err != nil {
		t.Fatalf("modules item --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var item canvas.ModuleItem
	if err := json.Unmarshal(dataJSON, &item); err != nil {
		t.Fatalf("data is not ModuleItem: %v", err)
	}
	if item.ID != "100" {
		t.Errorf("expected item ID '100', got %q", item.ID)
	}
	if item.Title != "Introduction" {
		t.Errorf("expected title 'Introduction', got %q", item.Title)
	}

	last := mock.LastRequest()
	if last.Path != "/api/v1/courses/1/modules/10/items/100" {
		t.Errorf("expected request to /api/v1/courses/1/modules/10/items/100, got %s", last.Path)
	}
}

func TestModulesItem_HumanMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	published := true
	mock.On("GET", "/api/v1/courses/1/modules/10/items/100", 200, map[string]any{
		"id":         "100",
		"module_id":  "10",
		"title":      "Introduction",
		"type":       "Page",
		"position":   1,
		"content_id": "50",
		"html_url":   "/courses/1/pages/introduction",
		"published":  published,
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	var buf bytes.Buffer
	cmd := newModulesItemCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("module", "10")

	err := cmd.RunE(cmd, []string{"100"})
	if err != nil {
		t.Fatalf("modules item failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Introduction") {
		t.Errorf("expected 'Introduction' in output, got: %s", output)
	}
	if !strings.Contains(output, "Page") {
		t.Errorf("expected type 'Page' in output, got: %s", output)
	}
	if !strings.Contains(output, "yes") {
		t.Errorf("expected 'yes' for published in output, got: %s", output)
	}
}
