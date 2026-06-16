package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
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

func TestCommandPath(t *testing.T) {
	root := &cobra.Command{Use: "canvas"}
	auth := &cobra.Command{Use: "auth"}
	login := &cobra.Command{Use: "login"}
	auth.AddCommand(login)
	root.AddCommand(auth)

	tests := []struct {
		name string
		cmd  *cobra.Command
		want string
	}{
		{"leaf command", login, "canvas auth login"},
		{"middle command", auth, "canvas auth"},
		{"root command", root, "canvas"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commandPath(tt.cmd)
			if got != tt.want {
				t.Errorf("commandPath(%q) = %q, want %q", tt.cmd.Name(), got, tt.want)
			}
		})
	}
}

func TestVersionCmd_JSONMode(t *testing.T) {
	cmd := NewRootCmd("3.0.0")
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"version", "--json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("version --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var info map[string]string
	if err := json.Unmarshal(dataJSON, &info); err != nil {
		t.Fatalf("data is not map: %v", err)
	}
	if info["version"] != "3.0.0" {
		t.Errorf("expected version '3.0.0', got %q", info["version"])
	}
}

func TestCompletionCmd_Bash(t *testing.T) {
	cmd := NewRootCmd("dev")
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"completion", "bash"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("completion bash failed: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Fatal("expected non-empty bash completion output")
	}
	if !strings.Contains(output, "bash") && !strings.Contains(output, "complete") {
		// Bash completions typically contain completion-related keywords
		t.Logf("bash completion output (first 200 chars): %s", output[:min(200, len(output))])
	}
}

func TestCompletionCmd_Zsh(t *testing.T) {
	cmd := NewRootCmd("dev")
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"completion", "zsh"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("completion zsh failed: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Fatal("expected non-empty zsh completion output")
	}
}

func TestCompletionCmd_Fish(t *testing.T) {
	cmd := NewRootCmd("dev")
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"completion", "fish"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("completion fish failed: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Fatal("expected non-empty fish completion output")
	}
}

func TestCompletionCmd_Powershell(t *testing.T) {
	cmd := NewRootCmd("dev")
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"completion", "powershell"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("completion powershell failed: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Fatal("expected non-empty powershell completion output")
	}
}

func TestCompletionCmd_InvalidShell(t *testing.T) {
	cmd := NewRootCmd("dev")
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"completion", "tcsh"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid shell")
	}
}

func TestExecute_Success(t *testing.T) {
	t.Setenv("CANVAS_BASE_URL", "https://test.instructure.com")
	t.Setenv("CANVAS_TOKEN", "test-token-123")

	code := Execute("1.0.0")
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestCommandSkipsFullConfig(t *testing.T) {
	// Build a command hierarchy matching the real CLI structure.
	root := &cobra.Command{Use: "canvas"}
	auth := &cobra.Command{Use: "auth"}
	login := &cobra.Command{Use: "login"}
	logout := &cobra.Command{Use: "logout"}
	status := &cobra.Command{Use: "status"}
	profiles := &cobra.Command{Use: "profiles"}
	use := &cobra.Command{Use: "use"}
	auth.AddCommand(login, logout, status, profiles, use)
	root.AddCommand(auth)

	versionCmd := &cobra.Command{Use: "version"}
	completionCmd := &cobra.Command{Use: "completion"}
	doctorCmd := &cobra.Command{Use: "doctor"}
	root.AddCommand(versionCmd, completionCmd, doctorCmd)

	courses := &cobra.Command{Use: "courses"}
	coursesList := &cobra.Command{Use: "list"}
	courses.AddCommand(coursesList)
	root.AddCommand(courses)

	tests := []struct {
		name string
		cmd  *cobra.Command
		want bool
	}{
		{"canvas version", versionCmd, true},
		{"canvas completion", completionCmd, true},
		{"canvas auth login", login, true},
		{"canvas auth logout", logout, true},
		{"canvas auth status", status, true},
		{"canvas auth profiles", profiles, true},
		{"canvas auth use", use, true},
		{"canvas doctor", doctorCmd, true},
		{"canvas courses list", coursesList, false},
		{"canvas auth", auth, false},
		{"canvas", root, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commandSkipsFullConfig(tt.cmd)
			if got != tt.want {
				t.Errorf("commandSkipsFullConfig(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
