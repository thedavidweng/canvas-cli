package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/config"
)

// lastConfig captures the ResolvedConfig from the most recent test subcommand execution.
var lastConfig *config.ResolvedConfig

// testSubCmd returns a cobra.Command that stores its config into lastConfig.
func testSubCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test-sub",
		Short: "test subcommand for capturing config",
		Run: func(cmd *cobra.Command, args []string) {
			lastConfig = GetConfig(cmd.Context())
		},
	}
}

func resetLastConfig() {
	lastConfig = nil
}

func TestRootCmd_HasCorrectMetadata(t *testing.T) {
	cmd := NewRootCmd("1.2.3")
	if cmd.Use != "canvas" {
		t.Errorf("expected Use 'canvas', got %q", cmd.Use)
	}
	if !strings.Contains(cmd.Short, "Canvas") {
		t.Errorf("expected Short to contain 'Canvas', got %q", cmd.Short)
	}
	if cmd.Long == "" {
		t.Error("expected non-empty Long description")
	}
}

func TestRootCmd_AllGlobalFlagsRegistered(t *testing.T) {
	cmd := NewRootCmd("dev")

	expectedFlags := []string{
		"json", "pretty", "compact", "ndjson", "full",
		"limit", "page-size", "no-paginate",
		"timeout", "retries",
		"dry-run", "confirm", "read-only",
		"events", "verbose", "debug", "quiet", "no-color",
		"confirm-delete",
		"config", "profile", "base-url",
	}

	flags := cmd.PersistentFlags()
	for _, name := range expectedFlags {
		if flags.Lookup(name) == nil {
			t.Errorf("missing persistent flag: --%s", name)
		}
	}
}

func TestRootCmd_PersistentPreRunE_LoadsConfig(t *testing.T) {
	resetLastConfig()
	t.Setenv("CANVAS_BASE_URL", "https://test.instructure.com")
	t.Setenv("CANVAS_TOKEN", "test-token-123")

	cmd := NewRootCmd("dev")
	cmd.AddCommand(testSubCmd())
	cmd.SetArgs([]string{"test-sub"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	if lastConfig == nil {
		t.Fatal("expected config to be set on context")
	}
	if lastConfig.BaseURL != "https://test.instructure.com" {
		t.Errorf("expected base URL 'https://test.instructure.com', got %q", lastConfig.BaseURL)
	}
	if lastConfig.Token != "test-token-123" {
		t.Errorf("expected token 'test-token-123', got %q", lastConfig.Token)
	}
}

func TestRootCmd_VersionSubcommand(t *testing.T) {
	cmd := NewRootCmd("2.0.0")
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"version"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "2.0.0") {
		t.Errorf("expected output to contain version '2.0.0', got %q", output)
	}
}

func TestRootCmd_CompletionSubcommand(t *testing.T) {
	cmd := NewRootCmd("dev")

	completionCmd, _, err := cmd.Find([]string{"completion"})
	if err != nil {
		t.Fatalf("completion subcommand not found: %v", err)
	}
	if completionCmd == nil {
		t.Fatal("expected completion subcommand to exist")
	}
}

func TestRootCmd_Flags_DefaultValues(t *testing.T) {
	cmd := NewRootCmd("dev")
	flags := cmd.PersistentFlags()

	if flags.Lookup("json").DefValue != "false" {
		t.Error("expected --json default false")
	}
	if flags.Lookup("pretty").DefValue != "false" {
		t.Error("expected --pretty default false")
	}
	if flags.Lookup("no-paginate").DefValue != "false" {
		t.Error("expected --no-paginate default false")
	}
	if flags.Lookup("dry-run").DefValue != "false" {
		t.Error("expected --dry-run default false")
	}
	if flags.Lookup("confirm").DefValue != "false" {
		t.Error("expected --confirm default false")
	}
	if flags.Lookup("read-only").DefValue != "false" {
		t.Error("expected --read-only default false")
	}
	if flags.Lookup("retries").DefValue != "3" {
		t.Errorf("expected --retries default 3, got %s", flags.Lookup("retries").DefValue)
	}
	if flags.Lookup("page-size").DefValue != "100" {
		t.Errorf("expected --page-size default 100, got %s", flags.Lookup("page-size").DefValue)
	}
	if flags.Lookup("limit").DefValue != "0" {
		t.Errorf("expected --limit default 0, got %s", flags.Lookup("limit").DefValue)
	}
}

func TestWithConfig_And_GetConfig(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "https://test.example.com",
		Token:   "tok",
		Profile: "default",
	}

	ctx := WithConfig(context.Background(), cfg)
	got := GetConfig(ctx)

	if got == nil {
		t.Fatal("expected non-nil config from context")
	}
	if got.BaseURL != "https://test.example.com" {
		t.Errorf("unexpected base URL: %s", got.BaseURL)
	}
}

func TestGetConfig_NilWhenMissing(t *testing.T) {
	got := GetConfig(context.Background())
	if got != nil {
		t.Errorf("expected nil config from empty context, got %+v", got)
	}
}
