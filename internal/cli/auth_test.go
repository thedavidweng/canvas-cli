package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
	"github.com/thedavidweng/canvas-cli/internal/testutil"
)

func TestAuthCmd_Exists(t *testing.T) {
	cmd := NewAuthCmd()
	if cmd.Use != "auth" {
		t.Errorf("expected Use 'auth', got %q", cmd.Use)
	}
}

func TestAuthCmd_HasSubcommands(t *testing.T) {
	cmd := NewAuthCmd()
	expected := []string{"status", "test", "login", "logout", "profiles", "use"}

	names := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("missing auth subcommand: %s", name)
		}
	}
}

func TestAuthStatus_ShowsProfileAndBaseURL(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "https://school.instructure.com",
		Token:   "secret-token-value",
		Profile: "default",
	}

	cmd := NewAuthCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Set up context with config
	subCmd, _, _ := cmd.Find([]string{"status"})
	if subCmd == nil {
		t.Fatal("status subcommand not found")
	}

	// We need to execute status with a config in context
	statusCmd := newAuthStatusCmd()
	statusCmd.SetContext(WithConfig(context.Background(), cfg))
	statusCmd.SetOut(&buf)

	err := statusCmd.RunE(statusCmd, nil)
	if err != nil {
		t.Fatalf("auth status failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "default") {
		t.Errorf("expected profile name in output, got: %s", output)
	}
	if !strings.Contains(output, "school.instructure.com") {
		t.Errorf("expected base URL in output, got: %s", output)
	}
	// Must never show the actual token
	if strings.Contains(output, "secret-token-value") {
		t.Errorf("auth status must never show the actual token value, got: %s", output)
	}
	// Should indicate token is present
	if !strings.Contains(output, "present") && !strings.Contains(output, "yes") && !strings.Contains(output, "Yes") {
		t.Errorf("expected token presence indication, got: %s", output)
	}
}

func TestAuthStatus_JSONMode(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "https://school.instructure.com",
		Token:   "secret-token-value",
		Profile: "default",
	}

	var buf bytes.Buffer
	statusCmd := newAuthStatusCmd()
	statusCmd.SetContext(WithConfig(context.Background(), cfg))
	statusCmd.SetOut(&buf)
	_ = statusCmd.Flags().Set("json", "true")

	err := statusCmd.RunE(statusCmd, nil)
	if err != nil {
		t.Fatalf("auth status failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}
	if env.Data == nil {
		t.Fatal("expected data in envelope")
	}
}

func TestAuthTest_CallsUserSelfEndpoint(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	testCmd := newAuthTestCmd()
	testCmd.SetContext(WithConfig(context.Background(), cfg))
	testCmd.SetOut(&buf)

	err := testCmd.RunE(testCmd, nil)
	if err != nil {
		t.Fatalf("auth test failed: %v", err)
	}

	// Verify the mock received a request to /api/v1/users/self
	if mock.RequestCount() == 0 {
		t.Fatal("expected at least one request to the mock server")
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

func TestAuthTest_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	testCmd := newAuthTestCmd()
	testCmd.SetContext(WithConfig(context.Background(), cfg))
	testCmd.SetOut(&buf)
	_ = testCmd.Flags().Set("json", "true")

	err := testCmd.RunE(testCmd, nil)
	if err != nil {
		t.Fatalf("auth test failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}
}

func TestAuthLogin_SavesToConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Start a mock server for the validation request
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	var buf bytes.Buffer
	loginCmd := newAuthLoginCmd()
	loginCmd.SetOut(&buf)
	_ = loginCmd.Flags().Set("base-url", mock.URL())
	_ = loginCmd.Flags().Set("config", configPath)
	_ = loginCmd.Flags().Set("token-stdin", "true")

	// Provide token via stdin
	tokenReader := strings.NewReader("my-new-token\n")
	loginCmd.SetIn(tokenReader)

	// We need to set context for the config resolution
	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "my-new-token",
		Profile: "default",
	}
	loginCmd.SetContext(WithConfig(context.Background(), cfg))

	err := loginCmd.RunE(loginCmd, []string{})
	if err != nil {
		t.Fatalf("auth login failed: %v", err)
	}

	// Verify config file was written
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config file not written: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, mock.URL()) {
		t.Errorf("expected base URL in config file, got: %s", content)
	}
}

func TestAuthLogout_RemovesToken(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write initial config
	initialConfig := `current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    token: old-token
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0600); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}

	var buf bytes.Buffer
	logoutCmd := newAuthLogoutCmd()
	logoutCmd.SetOut(&buf)
	_ = logoutCmd.Flags().Set("config", configPath)

	cfg := &config.ResolvedConfig{
		BaseURL: "https://school.instructure.com",
		Token:   "old-token",
		Profile: "default",
	}
	logoutCmd.SetContext(WithConfig(context.Background(), cfg))

	err := logoutCmd.RunE(logoutCmd, []string{})
	if err != nil {
		t.Fatalf("auth logout failed: %v", err)
	}

	// Verify token was removed from config
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	content := string(data)
	if strings.Contains(content, "old-token") {
		t.Errorf("expected token to be removed, config still contains it: %s", content)
	}
}

func TestAuthProfiles_ListsProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := `current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    token: env:CANVAS_TOKEN
  production:
    base_url: https://prod.instructure.com
    token: env:PROD_TOKEN
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	var buf bytes.Buffer
	profilesCmd := newAuthProfilesCmd()
	profilesCmd.SetOut(&buf)

	cfg := &config.ResolvedConfig{
		BaseURL: "https://school.instructure.com",
		Token:   "tok",
		Profile: "default",
	}
	profilesCmd.SetContext(WithConfig(context.Background(), cfg))
	_ = profilesCmd.Flags().Set("config", configPath)

	err := profilesCmd.RunE(profilesCmd, []string{})
	if err != nil {
		t.Fatalf("auth profiles failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "default") {
		t.Errorf("expected 'default' profile in output, got: %s", output)
	}
	if !strings.Contains(output, "production") {
		t.Errorf("expected 'production' profile in output, got: %s", output)
	}
}

func TestAuthUse_SwitchesProfile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := `current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    token: tok1
  production:
    base_url: https://prod.instructure.com
    token: tok2
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	var buf bytes.Buffer
	useCmd := newAuthUseCmd()
	useCmd.SetOut(&buf)
	_ = useCmd.Flags().Set("config", configPath)

	cfg := &config.ResolvedConfig{
		BaseURL: "https://school.instructure.com",
		Token:   "tok1",
		Profile: "default",
	}
	useCmd.SetContext(WithConfig(context.Background(), cfg))

	err := useCmd.RunE(useCmd, []string{"production"})
	if err != nil {
		t.Fatalf("auth use failed: %v", err)
	}

	// Verify current_profile was updated
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "current_profile: production") {
		t.Errorf("expected current_profile to be 'production', got: %s", content)
	}
}

func TestAuthTest_UnauthorizedResponse(t *testing.T) {
	// Create a mock server that returns 401
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{{"message": "Invalid access token"}},
		})
	}))
	defer server.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: server.URL,
		Token:   "bad-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	testCmd := newAuthTestCmd()
	testCmd.SetContext(WithConfig(context.Background(), cfg))
	testCmd.SetOut(&buf)

	err := testCmd.RunE(testCmd, nil)
	if err == nil {
		t.Fatal("expected error for unauthorized response")
	}
}
