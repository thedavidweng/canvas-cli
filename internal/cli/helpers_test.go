package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
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

	err := writeError(&buf, inputErr, "courses.list", true)
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

	err := writeError(&buf, inputErr, "courses.list", false)
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

func TestCheckSafety_Allowed(t *testing.T) {
	cfg := &config.ResolvedConfig{}
	err := checkSafety(cfg, false, true)
	if err != nil {
		t.Fatalf("expected allowed, got: %v", err)
	}
}

func TestCheckSafety_DryRun(t *testing.T) {
	cfg := &config.ResolvedConfig{}
	err := checkSafety(cfg, true, false)
	if err != nil {
		t.Fatalf("dry run should be allowed, got: %v", err)
	}
}

func TestCheckHighRiskSafety_Allowed(t *testing.T) {
	cfg := &config.ResolvedConfig{}
	err := checkHighRiskSafety(cfg, false, true)
	if err != nil {
		t.Fatalf("expected allowed, got: %v", err)
	}
}

func TestCheckHighRiskSafety_DryRun(t *testing.T) {
	cfg := &config.ResolvedConfig{}
	err := checkHighRiskSafety(cfg, true, false)
	if err != nil {
		t.Fatalf("dry run should be allowed, got: %v", err)
	}
}
