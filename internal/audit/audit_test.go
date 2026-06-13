package audit

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
)

// helper builds a minimal AuditEvent for testing.
func testEvent() canvas.AuditEvent {
	return canvas.AuditEvent{
		Time:           "2026-06-12T19:20:00Z",
		SchemaVersion:  "2026-06-12",
		Command:        "assignments.submit",
		Profile:        "default",
		BaseURL:        "https://school.instructure.com",
		Method:         "POST",
		Path:           "/api/v1/courses/123/assignments/456/submissions",
		Resource:       map[string]string{"course_id": "123", "assignment_id": "456"},
		RequestHash:    "sha256:abcdef1234567890",
		ResponseStatus: 200,
		DryRun:         false,
		Success:        true,
	}
}

func TestWriteEvent_AppendsJSONL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")

	a := NewAuditor(path, true)

	// Write two events.
	if err := a.WriteEvent(testEvent()); err != nil {
		t.Fatalf("first WriteEvent: %v", err)
	}
	ev2 := testEvent()
	ev2.Command = "grades.set"
	if err := a.WriteEvent(ev2); err != nil {
		t.Fatalf("second WriteEvent: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read audit file: %v", err)
	}

	lines := splitLines(string(data))
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	// Second line should have the second command.
	var got canvas.AuditEvent
	if err := json.Unmarshal([]byte(lines[1]), &got); err != nil {
		t.Fatalf("unmarshal second line: %v", err)
	}
	if got.Command != "grades.set" {
		t.Errorf("second event command = %q, want %q", got.Command, "grades.set")
	}
}

func TestWriteEvent_EachLineValidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")

	a := NewAuditor(path, true)

	for i := 0; i < 5; i++ {
		ev := testEvent()
		ev.Command = "test.command"
		if err := a.WriteEvent(ev); err != nil {
			t.Fatalf("WriteEvent %d: %v", i, err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read audit file: %v", err)
	}

	lines := splitLines(string(data))
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(lines))
	}

	for i, line := range lines {
		var raw json.RawMessage
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			t.Errorf("line %d is not valid JSON: %v\n  content: %s", i, err, line)
		}
	}
}

func TestWriteEvent_FieldsPresent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")

	a := NewAuditor(path, true)
	ev := testEvent()
	if err := a.WriteEvent(ev); err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read audit file: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	required := []string{"time", "command", "profile", "base_url", "method", "path", "resource", "response_status", "dry_run", "success"}
	for _, field := range required {
		if _, ok := got[field]; !ok {
			t.Errorf("missing required field %q in output", field)
		}
	}
}

func TestWriteEvent_TokenNeverInOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")

	a := NewAuditor(path, true)
	ev := testEvent()
	if err := a.WriteEvent(ev); err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read audit file: %v", err)
	}

	raw := string(data)
	if strings.Contains(strings.ToLower(raw), `"token"`) {
		t.Error("audit output contains 'token' field — tokens must never appear in audit logs")
	}
}

func TestWriteEvent_RequestBodyHashed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")

	a := NewAuditor(path, true)
	ev := testEvent()
	ev.RequestHash = HashBody(`{"submission":{"body":"my answer"}}`)
	if err := a.WriteEvent(ev); err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read audit file: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	hash, ok := got["request_hash"].(string)
	if !ok {
		t.Fatal("request_hash is not a string")
	}
	if !strings.HasPrefix(hash, "sha256:") {
		t.Errorf("request_hash = %q, want sha256: prefix", hash)
	}
	// Must not contain the raw body text.
	if strings.Contains(string(data), "my answer") {
		t.Error("raw request body leaked into audit log")
	}
}

func TestWriteEvent_FileCreatedIfMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent", "audit.jsonl")

	a := NewAuditor(path, true)
	if err := a.WriteEvent(testEvent()); err != nil {
		t.Fatalf("WriteEvent should create file and parent dirs: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("audit file was not created")
	}
}

func TestDefaultPath_XDGStateHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	got := DefaultPath()
	want := filepath.Join(dir, "canvas-cli", "audit.jsonl")
	if got != want {
		t.Errorf("DefaultPath() = %q, want %q", got, want)
	}
}

func TestDefaultPath_Fallback(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	got := DefaultPath()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".local", "state", "canvas-cli", "audit.jsonl")
	if got != want {
		t.Errorf("DefaultPath() = %q, want %q", got, want)
	}
}

func TestWriteEvent_DisabledNoOp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")

	a := NewAuditor(path, false)
	if err := a.WriteEvent(testEvent()); err != nil {
		t.Fatalf("WriteEvent on disabled auditor: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("disabled auditor should not create audit file")
	}
}

func TestHashBody_DeterministicSHA256(t *testing.T) {
	body := `{"key":"value"}`
	h1 := HashBody(body)
	h2 := HashBody(body)
	if h1 != h2 {
		t.Errorf("HashBody not deterministic: %q != %q", h1, h2)
	}
	if !strings.HasPrefix(h1, "sha256:") {
		t.Errorf("HashBody = %q, want sha256: prefix", h1)
	}
	// SHA256 hex is 64 chars; prefix adds 7 → 71 total.
	if len(h1) != 7+len(strings.TrimPrefix(h1, "sha256:")) {
		t.Errorf("unexpected hash length: %d", len(h1))
	}
}

func splitLines(s string) []string {
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
