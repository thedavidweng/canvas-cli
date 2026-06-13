package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
)

func TestWriteJSON_OutputsValidJSON(t *testing.T) {
	env := NewSuccess([]string{"a", "b"}, "courses.list")
	var buf bytes.Buffer

	if err := WriteJSON(&buf, env, false); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw: %s", err, buf.String())
	}
}

func TestWriteJSON_CompactMode(t *testing.T) {
	env := NewSuccess(map[string]string{"key": "value"}, "courses.list")
	var buf bytes.Buffer

	if err := WriteJSON(&buf, env, false); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	output := buf.String()
	// Compact mode should be a single line (no newlines except trailing)
	trimmed := strings.TrimRight(output, "\n")
	if strings.Contains(trimmed, "\n") {
		t.Errorf("compact output contains newlines:\n%s", output)
	}
}

func TestWriteJSON_PrettyMode(t *testing.T) {
	env := NewSuccess(map[string]string{"key": "value"}, "courses.list")
	var buf bytes.Buffer

	if err := WriteJSON(&buf, env, true); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "\n") {
		t.Error("pretty output should contain newlines")
	}
	if !strings.Contains(output, "  ") {
		t.Error("pretty output should contain indentation")
	}
}

func TestWriteJSON_NeverContainsToken(t *testing.T) {
	secretToken := "super-secret-token-12345"
	env := NewSuccess(nil, "courses.list")
	env.Meta.Profile = secretToken

	var buf bytes.Buffer
	if err := WriteJSON(&buf, env, true); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	output := buf.String()
	// The Envelope/Meta struct does not contain a Token field.
	// Verify by parsing and checking the structure does not leak tokens.
	var parsed map[string]any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Walk the JSON to check no "token" key exists anywhere
	assertNoToken(t, parsed, output)
}

func assertNoToken(t *testing.T, v any, raw string) {
	t.Helper()
	switch val := v.(type) {
	case map[string]any:
		for k, vv := range val {
			if strings.EqualFold(k, "token") {
				t.Errorf("JSON output contains key %q", k)
			}
			assertNoToken(t, vv, raw)
		}
	case []any:
		for _, vv := range val {
			assertNoToken(t, vv, raw)
		}
	}
}

func TestWriteJSON_ErrorEnvelope(t *testing.T) {
	errInfo := canvas.ErrorInfo{
		Code:     "CANVAS_API_ERROR",
		Message:  "bad request",
		Category: "api",
		Status:   400,
	}
	env := NewError(errInfo, "assignments.list")
	var buf bytes.Buffer

	if err := WriteJSON(&buf, env, false); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if parsed["ok"] != false {
		t.Errorf("ok = %v, want false", parsed["ok"])
	}
	if parsed["error"] == nil {
		t.Fatal("expected error field")
	}
}

func TestWriteJSON_PreservesAllMetaFields(t *testing.T) {
	rl := &canvas.RateLimit{RequestCost: 1.0, Remaining: 999}
	overrides := canvas.Meta{
		Profile:      "prod",
		BaseURL:      "https://canvas.example.com",
		DurationMS:   50,
		RequestCount: 2,
		Paginated:    true,
		PageSize:     50,
		RateLimit:    rl,
		Warnings:     []string{"slow response"},
	}
	env := NewSuccess([]int{}, "modules.list", overrides)
	var buf bytes.Buffer

	if err := WriteJSON(&buf, env, true); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	meta, ok := parsed["meta"].(map[string]any)
	if !ok {
		t.Fatal("meta is not an object")
	}
	if meta["schema_version"] != SchemaVersion {
		t.Errorf("meta.schema_version = %v, want %q", meta["schema_version"], SchemaVersion)
	}
	if meta["command"] != "modules.list" {
		t.Errorf("meta.command = %v, want %q", meta["command"], "modules.list")
	}
	if meta["profile"] != "prod" {
		t.Errorf("meta.profile = %v, want prod", meta["profile"])
	}
	if meta["duration_ms"].(float64) != 50 {
		t.Errorf("meta.duration_ms = %v, want 50", meta["duration_ms"])
	}
	if meta["rate_limit"] == nil {
		t.Error("expected rate_limit in meta")
	}
	if warnings, ok := meta["warnings"].([]any); !ok || len(warnings) != 1 {
		t.Errorf("meta.warnings = %v, want [slow response]", meta["warnings"])
	}
}
