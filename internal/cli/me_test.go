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

func TestMeCmd_Exists(t *testing.T) {
	cmd := NewMeCmd()
	if cmd.Use != "me" {
		t.Errorf("expected Use 'me', got %q", cmd.Use)
	}
}

func TestMeCmd_HasGetSubcommand(t *testing.T) {
	cmd := NewMeCmd()
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

func TestMeGet_CallsAPI(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("me get failed: %v", err)
	}

	// Verify the mock received a request to /api/v1/users/self
	if mock.RequestCount() == 0 {
		t.Fatal("expected at least one request")
	}

	last := mock.LastRequest()
	if last.Path != "/api/v1/users/self" {
		t.Errorf("expected request to /api/v1/users/self, got %s", last.Path)
	}

	output := buf.String()
	if !strings.Contains(output, "Test User") {
		t.Errorf("expected user name in output, got: %s", output)
	}
}

func TestMeGet_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("me get failed: %v", err)
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

func TestMeGet_ShowsUserInfo(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("me get failed: %v", err)
	}

	output := buf.String()
	// Should show user info
	if !strings.Contains(output, "Test User") {
		t.Errorf("expected user name, got: %s", output)
	}
	if !strings.Contains(output, "1") {
		t.Errorf("expected user ID, got: %s", output)
	}
}

func TestMeGet_APIError(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	// Override the /api/v1/users/self to return 401
	mock.On("GET", "/api/v1/users/self", 401, map[string]any{
		"errors": []map[string]any{{"message": "Unauthorized"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "bad-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for unauthorized response")
	}
}
