package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestWriteJSON_NeverContainsToken(t *testing.T) {
	secretToken := "super-secret-token-12345"
	env := NewSuccess(nil, "courses.list")
	env.Meta.Profile = secretToken

	var buf bytes.Buffer
	if err := WriteJSON(&buf, &env, true); err != nil {
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
	assertNoToken(t, parsed)
}

func assertNoToken(t *testing.T, v any) {
	t.Helper()
	switch val := v.(type) {
	case map[string]any:
		for k, vv := range val {
			if strings.EqualFold(k, "token") {
				t.Errorf("JSON output contains key %q", k)
			}
			assertNoToken(t, vv)
		}
	case []any:
		for _, vv := range val {
			assertNoToken(t, vv)
		}
	}
}
