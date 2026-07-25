package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
	"github.com/thedavidweng/canvas-cli/internal/safety"
)

func TestGetClientFromContext_NilConfig(t *testing.T) {
	ctx := context.Background()
	_, err := getClientFromContext(ctx)
	if err == nil {
		t.Fatal("expected error when config is nil, got nil")
	}
	if err.Error() != "no config loaded" {
		t.Errorf("expected 'no config loaded', got %q", err.Error())
	}
}

func TestGetClientFromContext_ValidConfig(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "https://canvas.example.com",
		Token:   "tok123",
	}
	ctx := WithConfig(context.Background(), cfg)
	client, err := getClientFromContext(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestWriteOutput_JSONMode(t *testing.T) {
	cfg := &config.ResolvedConfig{
		Profile: "test",
		BaseURL: "https://canvas.example.com",
	}
	data := map[string]string{"id": "1", "name": "Test"}
	var buf bytes.Buffer

	err := writeOutput(&buf, cfg, data, "courses.list", true)
	if err != nil {
		t.Fatalf("writeOutput in JSON mode failed: %v", err)
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
	if env.Meta.Command != "courses.list" {
		t.Errorf("expected command 'courses.list', got %q", env.Meta.Command)
	}
	if env.Meta.Profile != "test" {
		t.Errorf("expected profile 'test', got %q", env.Meta.Profile)
	}
}

func TestWriteOutput_HumanMode_NoFn(t *testing.T) {
	cfg := &config.ResolvedConfig{Profile: "test"}
	var buf bytes.Buffer

	err := writeOutput(&buf, cfg, nil, "courses.list", false)
	if err != nil {
		t.Fatalf("writeOutput in human mode failed: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output, got %q", buf.String())
	}
}

func TestWriteOutput_HumanMode_WithFn(t *testing.T) {
	cfg := &config.ResolvedConfig{Profile: "test"}
	var buf bytes.Buffer

	err := writeOutput(&buf, cfg, nil, "courses.list", false, func(w io.Writer) error {
		_, err := w.Write([]byte("human output"))
		return err
	})
	if err != nil {
		t.Fatalf("writeOutput with humanFn failed: %v", err)
	}
	if buf.String() != "human output" {
		t.Errorf("expected 'human output', got %q", buf.String())
	}
}

func TestWriteError_JSONMode(t *testing.T) {
	var buf bytes.Buffer
	inputErr := errors.New("something went wrong")

	err := writeError(&buf, &config.ResolvedConfig{}, inputErr, "courses.list", true)
	if err != nil {
		t.Fatalf("writeError in JSON mode returned error: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if env.OK {
		t.Error("expected ok:false in error envelope")
	}
	if env.Error == nil {
		t.Fatal("expected error in envelope")
	}
	if env.Error.Message != "something went wrong" {
		t.Errorf("expected error message 'something went wrong', got %q", env.Error.Message)
	}
}

func TestWriteError_HumanMode(t *testing.T) {
	var buf bytes.Buffer
	inputErr := errors.New("something went wrong")

	err := writeError(&buf, &config.ResolvedConfig{}, inputErr, "courses.list", false)
	if err == nil {
		t.Fatal("expected error to be returned")
	}
	if err.Error() != "something went wrong" {
		t.Errorf("expected 'something went wrong', got %q", err.Error())
	}
}

func TestIsJSONMode(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("json", false, "")

	if isJSONMode(cmd) {
		t.Error("expected false when --json not set")
	}

	_ = cmd.Flags().Set("json", "true")
	if !isJSONMode(cmd) {
		t.Error("expected true when --json is set")
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
	}
	for _, tt := range tests {
		got := truncateString(tt.s, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
		}
	}
}

func TestExitError_Error(t *testing.T) {
	e := &exitError{msg: "something broke", exitCode: 42}
	if e.Error() != "something broke" {
		t.Errorf("Error() = %q, want %q", e.Error(), "something broke")
	}
}

func TestExitError_ExitCode(t *testing.T) {
	e := &exitError{msg: "blocked", exitCode: 1}
	if e.ExitCode() != 1 {
		t.Errorf("ExitCode() = %d, want 1", e.ExitCode())
	}

	e2 := &exitError{msg: "ok", exitCode: 0}
	if e2.ExitCode() != 0 {
		t.Errorf("ExitCode() = %d, want 0", e2.ExitCode())
	}

	e3 := &exitError{msg: "fatal", exitCode: 255}
	if e3.ExitCode() != 255 {
		t.Errorf("ExitCode() = %d, want 255", e3.ExitCode())
	}
}

func TestExitError_ExitCodePartialFailure(t *testing.T) {
	err := &exitError{msg: "partial failure", exitCode: 8}
	if err.ExitCode() != 8 {
		t.Errorf("expected exit code 8, got %d", err.ExitCode())
	}
}

func TestWriteNetworkError_JSONMode(t *testing.T) {
	var buf bytes.Buffer
	inputErr := errors.New("connection refused")

	err := writeNetworkError(&buf, &config.ResolvedConfig{}, inputErr, "api.get", true)
	if err != nil {
		t.Fatalf("writeNetworkError in JSON mode returned error: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if env.OK {
		t.Error("expected ok:false in error envelope")
	}
	if env.Error == nil {
		t.Fatal("expected error in envelope")
	}
	if env.Error.Code != "CANVAS_NETWORK_ERROR" {
		t.Errorf("expected code CANVAS_NETWORK_ERROR, got %q", env.Error.Code)
	}
	if env.Error.Category != "network" {
		t.Errorf("expected category 'network', got %q", env.Error.Category)
	}
}

func TestWriteNetworkError_HumanMode(t *testing.T) {
	var buf bytes.Buffer
	inputErr := errors.New("connection refused")

	err := writeNetworkError(&buf, &config.ResolvedConfig{}, inputErr, "api.get", false)
	if err == nil {
		t.Fatal("expected error to be returned")
	}
	if err.Error() != "connection refused" {
		t.Errorf("expected 'connection refused', got %q", err.Error())
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output in human mode, got %q", buf.String())
	}
}

func TestWriteErrorWithCode_JSONMode(t *testing.T) {
	var buf bytes.Buffer
	inputErr := errors.New("not found")

	err := writeErrorWithCode(&buf, &config.ResolvedConfig{}, inputErr, "courses.get", "CANVAS_NOT_FOUND", "api", true)
	if err != nil {
		t.Fatalf("writeErrorWithCode in JSON mode returned error: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if env.Error == nil {
		t.Fatal("expected error in envelope")
	}
	if env.Error.Code != "CANVAS_NOT_FOUND" {
		t.Errorf("expected code CANVAS_NOT_FOUND, got %q", env.Error.Code)
	}
}

func TestWriteErrorWithCode_HumanMode(t *testing.T) {
	var buf bytes.Buffer
	inputErr := errors.New("not found")

	err := writeErrorWithCode(&buf, &config.ResolvedConfig{}, inputErr, "courses.get", "CANVAS_NOT_FOUND", "api", false)
	if err == nil {
		t.Fatal("expected error to be returned")
	}
	if err.Error() != "not found" {
		t.Errorf("expected 'not found', got %q", err.Error())
	}
}

func TestCheckSafetyLevel_BlockedByReadOnly(t *testing.T) {
	cfg := &config.ResolvedConfig{ReadOnly: true}
	err := checkSafetyLevel(cfg, false, false, safety.LowRiskWrite)
	if err == nil {
		t.Fatal("expected error when read-only and no confirm")
	}
	exitErr, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected *exitError, got %T", err)
	}
	if exitErr.ExitCode() == 0 {
		t.Error("expected non-zero exit code for blocked operation")
	}
}

func TestCheckSafetyLevel_HighRiskBlocked(t *testing.T) {
	cfg := &config.ResolvedConfig{}
	err := checkSafetyLevel(cfg, false, false, safety.HighRiskWrite)
	if err == nil {
		t.Fatal("expected error for high-risk write without confirm")
	}
}

func TestCheckSafetyLevel_AllowedWithConfirm(t *testing.T) {
	cfg := &config.ResolvedConfig{}
	err := checkSafetyLevel(cfg, false, true, safety.HighRiskWrite)
	if err != nil {
		t.Fatalf("expected allowed with confirm, got: %v", err)
	}
}

func TestWriteAudit_DisabledNoOp(t *testing.T) {
	cfg := &config.ResolvedConfig{AuditEnabled: false}
	// Should not panic or write anything
	writeAudit(cfg, "test.cmd", "POST", "/path", "{}", false, 200, true)
}

func TestWriteAuditWithResource_WritesEvent(t *testing.T) {
	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
	cfg := &config.ResolvedConfig{
		Profile:      "test",
		BaseURL:      "https://canvas.example.com",
		AuditEnabled: true,
		AuditPath:    auditPath,
	}

	resource := map[string]string{
		"course_id":     "1",
		"assignment_id": "100",
	}
	writeAuditWithResource(cfg, "assignments.submit", "POST", "/api/v1/courses/1/assignments/100/submissions",
		`{"submission":"test"}`, false, 201, true, resource)

	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected audit event to be written")
	}

	var event canvas.AuditEvent
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("failed to parse audit event: %v", err)
	}
	if event.Command != "assignments.submit" {
		t.Errorf("expected command 'assignments.submit', got %q", event.Command)
	}
	if event.ResponseStatus != 201 {
		t.Errorf("expected response status 201, got %d", event.ResponseStatus)
	}
	if !event.Success {
		t.Error("expected success=true")
	}
	if event.Resource["course_id"] != "1" {
		t.Errorf("expected resource course_id '1', got %q", event.Resource["course_id"])
	}
	if event.Resource["assignment_id"] != "100" {
		t.Errorf("expected resource assignment_id '100', got %q", event.Resource["assignment_id"])
	}
}

func TestWriteAuditWithResource_NilResourceDefaultsToEmpty(t *testing.T) {
	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
	cfg := &config.ResolvedConfig{
		Profile:      "test",
		BaseURL:      "https://canvas.example.com",
		AuditEnabled: true,
		AuditPath:    auditPath,
	}

	writeAuditWithResource(cfg, "test.cmd", "POST", "/path", "", false, 200, true, nil)

	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit file: %v", err)
	}

	var event canvas.AuditEvent
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("failed to parse audit event: %v", err)
	}
	if event.Resource == nil {
		t.Fatal("expected non-nil resource map")
	}
	if len(event.Resource) != 0 {
		t.Errorf("expected empty resource map, got %v", event.Resource)
	}
}

func TestWriteAuditWithResource_FailureRecorded(t *testing.T) {
	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
	cfg := &config.ResolvedConfig{
		Profile:      "test",
		BaseURL:      "https://canvas.example.com",
		AuditEnabled: true,
		AuditPath:    auditPath,
	}

	writeAuditWithResource(cfg, "api.post", "POST", "/api/v1/test", "", false, 403, false, nil)

	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit file: %v", err)
	}

	var event canvas.AuditEvent
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("failed to parse audit event: %v", err)
	}
	if event.ResponseStatus != 403 {
		t.Errorf("expected response status 403, got %d", event.ResponseStatus)
	}
	if event.Success {
		t.Error("expected success=false for failed request")
	}
}

func TestTruncateString_Multibyte(t *testing.T) {
	// Chinese characters - each is 3 bytes in UTF-8 but 1 rune
	s := "你好世界测试"
	got := truncateString(s, 4)
	if !strings.HasSuffix(got, "...") {
		t.Errorf("expected truncation suffix, got %q", got)
	}
	// Should keep first 4 runes + "..."
	want := "你好世界..."
	if got != want {
		t.Errorf("truncateString(%q, 4) = %q, want %q", s, got, want)
	}
}

func TestTruncateString_NoSplitMidRune(t *testing.T) {
	// Emoji is 4 bytes in UTF-8, but rune-aware truncation keeps it intact
	s := "a😀b"
	got := truncateString(s, 2)
	// Should be "a😀..." not a split emoji byte sequence
	want := "a😀..."
	if got != want {
		t.Errorf("truncateString(%q, 2) = %q, want %q", s, got, want)
	}
}
