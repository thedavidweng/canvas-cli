package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

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
	// Should indicate token auth method
	if !strings.Contains(output, "Auth:       token") {
		t.Errorf("expected token auth method in output, got: %s", output)
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

func TestAuthStatus_ShowsCookie(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "https://school.instructure.com",
		Token:   "",
		Cookie:  "session-cookie-value",
		Profile: "default",
	}

	var buf bytes.Buffer
	statusCmd := newAuthStatusCmd()
	statusCmd.SetContext(WithConfig(context.Background(), cfg))
	statusCmd.SetOut(&buf)

	err := statusCmd.RunE(statusCmd, nil)
	if err != nil {
		t.Fatalf("auth status failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Auth:       cookie (experimental)") {
		t.Errorf("expected cookie auth method in output, got: %s", output)
	}
	// Must never show the actual cookie value
	if strings.Contains(output, "session-cookie-value") {
		t.Errorf("auth status must never show the actual cookie value, got: %s", output)
	}
}

func TestAuthStatus_NoCookie(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "https://school.instructure.com",
		Token:   "some-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	statusCmd := newAuthStatusCmd()
	statusCmd.SetContext(WithConfig(context.Background(), cfg))
	statusCmd.SetOut(&buf)

	err := statusCmd.RunE(statusCmd, nil)
	if err != nil {
		t.Fatalf("auth status failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Auth:       token") {
		t.Errorf("expected token auth method in output, got: %s", output)
	}
}

func TestAuthStatus_JSONMode_ShowsCookiePresent(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "https://school.instructure.com",
		Token:   "",
		Cookie:  "session-cookie-value",
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

	data, ok := env.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected data to be object, got %T", env.Data)
	}
	if data["cookie_present"] != true {
		t.Errorf("expected cookie_present=true, got %v", data["cookie_present"])
	}
	if data["token_present"] != false {
		t.Errorf("expected token_present=false, got %v", data["token_present"])
	}
	if data["auth_method"] != "cookie (experimental)" {
		t.Errorf("expected auth_method='cookie (experimental)', got %v", data["auth_method"])
	}
}

func TestAuthTest_WithCookieAuth(t *testing.T) {
	// Create a mock that checks for cookie header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "1",
			"name": "Cookie User",
		})
	}))
	defer server.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: server.URL,
		Token:   "",
		Cookie:  "test-session-cookie",
		Profile: "default",
	}

	var buf bytes.Buffer
	testCmd := newAuthTestCmd()
	testCmd.SetContext(WithConfig(context.Background(), cfg))
	testCmd.SetOut(&buf)

	err := testCmd.RunE(testCmd, nil)
	if err != nil {
		t.Fatalf("auth test with cookie failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Cookie User") {
		t.Errorf("expected user name in output, got: %s", output)
	}
}

func TestAuthLogin_CookieStdin(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	mock := testutil.NewMockCanvas()
	defer mock.Close()

	var buf bytes.Buffer
	loginCmd := newAuthLoginCmd()
	loginCmd.SetOut(&buf)
	_ = loginCmd.Flags().Set("base-url", mock.URL())
	_ = loginCmd.Flags().Set("config", configPath)
	_ = loginCmd.Flags().Set("cookie-stdin", "true")

	cookieReader := strings.NewReader("my-session-cookie\n")
	loginCmd.SetIn(cookieReader)

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Profile: "default",
	}
	loginCmd.SetContext(WithConfig(context.Background(), cfg))

	err := loginCmd.RunE(loginCmd, []string{})
	if err != nil {
		t.Fatalf("auth login with cookie-stdin failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config file not written: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "my-session-cookie") {
		t.Errorf("expected cookie in config file, got: %s", content)
	}
}

func TestAuthLogin_CookieStdin_WithCSRF(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	var buf bytes.Buffer
	loginCmd := newAuthLoginCmd()
	loginCmd.SetOut(&buf)
	_ = loginCmd.Flags().Set("base-url", "https://school.instructure.com")
	_ = loginCmd.Flags().Set("config", configPath)
	_ = loginCmd.Flags().Set("cookie-stdin", "true")
	_ = loginCmd.Flags().Set("csrf-token-stdin", "true")

	// Provide both cookie and CSRF on stdin (one line each)
	inputReader := strings.NewReader("my-session-cookie\nmy-csrf-token\n")
	loginCmd.SetIn(inputReader)

	cfg := &config.ResolvedConfig{
		BaseURL: "https://school.instructure.com",
		Profile: "default",
	}
	loginCmd.SetContext(WithConfig(context.Background(), cfg))

	err := loginCmd.RunE(loginCmd, []string{})
	if err != nil {
		t.Fatalf("auth login failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config file not written: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "my-session-cookie") {
		t.Errorf("expected cookie in config, got: %s", content)
	}
	if !strings.Contains(content, "my-csrf-token") {
		t.Errorf("expected csrf_token in config, got: %s", content)
	}
}

func TestAuthLogin_CookieEnv(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	t.Setenv("MY_COOKIE", "env-cookie-value")

	var buf bytes.Buffer
	loginCmd := newAuthLoginCmd()
	loginCmd.SetOut(&buf)
	_ = loginCmd.Flags().Set("base-url", "https://school.instructure.com")
	_ = loginCmd.Flags().Set("config", configPath)
	_ = loginCmd.Flags().Set("cookie-env", "MY_COOKIE")

	cfg := &config.ResolvedConfig{
		BaseURL: "https://school.instructure.com",
		Profile: "default",
	}
	loginCmd.SetContext(WithConfig(context.Background(), cfg))

	err := loginCmd.RunE(loginCmd, []string{})
	if err != nil {
		t.Fatalf("auth login with cookie-env failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config file not written: %v", err)
	}

	content := string(data)
	// Should store as env:MY_COOKIE (same pattern as token-env)
	if !strings.Contains(content, "env:MY_COOKIE") {
		t.Errorf("expected 'env:MY_COOKIE' in config, got: %s", content)
	}
}

func TestAuthLogin_CSRFTokenEnv(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	t.Setenv("MY_CSRF", "csrf-value")
	t.Setenv("MY_COOKIE", "cookie-value")

	var buf bytes.Buffer
	loginCmd := newAuthLoginCmd()
	loginCmd.SetOut(&buf)
	_ = loginCmd.Flags().Set("base-url", "https://school.instructure.com")
	_ = loginCmd.Flags().Set("config", configPath)
	_ = loginCmd.Flags().Set("cookie-env", "MY_COOKIE")
	_ = loginCmd.Flags().Set("csrf-token-env", "MY_CSRF")

	cfg := &config.ResolvedConfig{
		BaseURL: "https://school.instructure.com",
		Profile: "default",
	}
	loginCmd.SetContext(WithConfig(context.Background(), cfg))

	err := loginCmd.RunE(loginCmd, []string{})
	if err != nil {
		t.Fatalf("auth login failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config file not written: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "env:MY_CSRF") {
		t.Errorf("expected 'env:MY_CSRF' in config, got: %s", content)
	}
}

func TestAuthLogin_CookieFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	cookieFilePath := filepath.Join(tmpDir, "cookie.txt")

	if err := os.WriteFile(cookieFilePath, []byte("file-cookie-value\n"), 0600); err != nil {
		t.Fatalf("failed to write cookie file: %v", err)
	}

	var buf bytes.Buffer
	loginCmd := newAuthLoginCmd()
	loginCmd.SetOut(&buf)
	_ = loginCmd.Flags().Set("base-url", "https://school.instructure.com")
	_ = loginCmd.Flags().Set("config", configPath)
	_ = loginCmd.Flags().Set("cookie-file", cookieFilePath)

	cfg := &config.ResolvedConfig{
		BaseURL: "https://school.instructure.com",
		Profile: "default",
	}
	loginCmd.SetContext(WithConfig(context.Background(), cfg))

	err := loginCmd.RunE(loginCmd, []string{})
	if err != nil {
		t.Fatalf("auth login with cookie-file failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config file not written: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "file-cookie-value") {
		t.Errorf("expected cookie value in config, got: %s", content)
	}
	// File path must not appear in output
	output := buf.String()
	if strings.Contains(output, cookieFilePath) {
		t.Errorf("cookie file path should be redacted from output, got: %s", output)
	}
}

func TestAuthLogin_CookieFile_BadPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file permissions not enforced on Windows")
	}
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	cookieFilePath := filepath.Join(tmpDir, "cookie.txt")

	// Create file with world-readable permissions
	if err := os.WriteFile(cookieFilePath, []byte("secret-cookie"), 0644); err != nil {
		t.Fatalf("failed to write cookie file: %v", err)
	}

	var buf bytes.Buffer
	loginCmd := newAuthLoginCmd()
	loginCmd.SetOut(&buf)
	_ = loginCmd.Flags().Set("base-url", "https://school.instructure.com")
	_ = loginCmd.Flags().Set("config", configPath)
	_ = loginCmd.Flags().Set("cookie-file", cookieFilePath)

	cfg := &config.ResolvedConfig{
		BaseURL: "https://school.instructure.com",
		Profile: "default",
	}
	loginCmd.SetContext(WithConfig(context.Background(), cfg))

	err := loginCmd.RunE(loginCmd, []string{})
	if err == nil {
		t.Fatal("expected error for world-readable cookie file")
	}
	if !strings.Contains(err.Error(), "too-permissive") {
		t.Errorf("expected permission error, got: %v", err)
	}
}

func TestAuthLogin_CSRFTokenFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	cookieFilePath := filepath.Join(tmpDir, "cookie.txt")
	csrfFilePath := filepath.Join(tmpDir, "csrf.txt")

	if err := os.WriteFile(cookieFilePath, []byte("session-cookie\n"), 0600); err != nil {
		t.Fatalf("failed to write cookie file: %v", err)
	}
	if err := os.WriteFile(csrfFilePath, []byte("csrf-token-value\n"), 0600); err != nil {
		t.Fatalf("failed to write csrf file: %v", err)
	}

	var buf bytes.Buffer
	loginCmd := newAuthLoginCmd()
	loginCmd.SetOut(&buf)
	_ = loginCmd.Flags().Set("base-url", "https://school.instructure.com")
	_ = loginCmd.Flags().Set("config", configPath)
	_ = loginCmd.Flags().Set("cookie-file", cookieFilePath)
	_ = loginCmd.Flags().Set("csrf-token-file", csrfFilePath)

	cfg := &config.ResolvedConfig{
		BaseURL: "https://school.instructure.com",
		Profile: "default",
	}
	loginCmd.SetContext(WithConfig(context.Background(), cfg))

	err := loginCmd.RunE(loginCmd, []string{})
	if err != nil {
		t.Fatalf("auth login failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config file not written: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "csrf-token-value") {
		t.Errorf("expected csrf token in config, got: %s", content)
	}
}

func TestAuthLogin_CSRFTokenFile_BadPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file permissions not enforced on Windows")
	}
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	cookieFilePath := filepath.Join(tmpDir, "cookie.txt")
	csrfFilePath := filepath.Join(tmpDir, "csrf.txt")

	if err := os.WriteFile(cookieFilePath, []byte("session-cookie\n"), 0600); err != nil {
		t.Fatalf("failed to write cookie file: %v", err)
	}
	// CSRF file with group-readable permissions
	if err := os.WriteFile(csrfFilePath, []byte("csrf-token-value"), 0640); err != nil {
		t.Fatalf("failed to write csrf file: %v", err)
	}

	var buf bytes.Buffer
	loginCmd := newAuthLoginCmd()
	loginCmd.SetOut(&buf)
	_ = loginCmd.Flags().Set("base-url", "https://school.instructure.com")
	_ = loginCmd.Flags().Set("config", configPath)
	_ = loginCmd.Flags().Set("cookie-file", cookieFilePath)
	_ = loginCmd.Flags().Set("csrf-token-file", csrfFilePath)

	cfg := &config.ResolvedConfig{
		BaseURL: "https://school.instructure.com",
		Profile: "default",
	}
	loginCmd.SetContext(WithConfig(context.Background(), cfg))

	err := loginCmd.RunE(loginCmd, []string{})
	if err == nil {
		t.Fatal("expected error for group-readable CSRF file")
	}
	if !strings.Contains(err.Error(), "too-permissive") {
		t.Errorf("expected permission error, got: %v", err)
	}
}

func TestAuthLogout_ClearsCookie(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := `current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    token: old-token
    cookie: old-session-cookie
    csrf_token: old-csrf-token
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
		Cookie:  "old-session-cookie",
		Profile: "default",
	}
	logoutCmd.SetContext(WithConfig(context.Background(), cfg))

	err := logoutCmd.RunE(logoutCmd, []string{})
	if err != nil {
		t.Fatalf("auth logout failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	content := string(data)
	if strings.Contains(content, "old-token") {
		t.Errorf("expected token to be removed, config still contains it: %s", content)
	}
	if strings.Contains(content, "old-session-cookie") {
		t.Errorf("expected cookie to be removed, config still contains it: %s", content)
	}
	if strings.Contains(content, "old-csrf-token") {
		t.Errorf("expected csrf_token to be removed, config still contains it: %s", content)
	}
}

func TestReadSecretFile_ValidPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	secretPath := filepath.Join(tmpDir, "secret.txt")

	if err := os.WriteFile(secretPath, []byte("  secret-value  \n"), 0600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	val, err := readSecretFile(secretPath)
	if err != nil {
		t.Fatalf("readSecretFile failed: %v", err)
	}
	if val != "secret-value" {
		t.Errorf("expected trimmed value 'secret-value', got %q", val)
	}
}

func TestReadSecretFile_TooPermissive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file permissions not enforced on Windows")
	}
	tmpDir := t.TempDir()
	secretPath := filepath.Join(tmpDir, "secret.txt")

	if err := os.WriteFile(secretPath, []byte("secret"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err := readSecretFile(secretPath)
	if err == nil {
		t.Fatal("expected error for too-permissive file")
	}
	if !strings.Contains(err.Error(), "too-permissive") {
		t.Errorf("expected permission error, got: %v", err)
	}
}

func TestReadSecretFile_GroupReadable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file permissions not enforced on Windows")
	}
	tmpDir := t.TempDir()
	secretPath := filepath.Join(tmpDir, "secret.txt")

	if err := os.WriteFile(secretPath, []byte("secret"), 0640); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err := readSecretFile(secretPath)
	if err == nil {
		t.Fatal("expected error for group-readable file")
	}
}

func TestReadSecretFile_WorldReadable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file permissions not enforced on Windows")
	}
	tmpDir := t.TempDir()
	secretPath := filepath.Join(tmpDir, "secret.txt")

	if err := os.WriteFile(secretPath, []byte("secret"), 0604); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err := readSecretFile(secretPath)
	if err == nil {
		t.Fatal("expected error for world-readable file")
	}
}

func TestReadSecretFile_FileNotFound(t *testing.T) {
	_, err := readSecretFile("/nonexistent/path/secret.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestExtractHost(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://school.instructure.com", "school.instructure.com"},
		{"https://canvas.school.edu:8080", "canvas.school.edu"},
		{"http://localhost:3000", "localhost"},
		{"https://school.instructure.com/api/v1", "school.instructure.com"},
		{"not-a-url", ""},
	}
	for _, tt := range tests {
		got := extractHost(tt.input)
		if got != tt.want {
			t.Errorf("extractHost(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAuthLogin_FlagsExist(t *testing.T) {
	loginCmd := newAuthLoginCmd()

	expectedFlags := []string{
		"cookie-stdin", "cookie-env", "cookie-file",
		"csrf-token-stdin", "csrf-token-env", "csrf-token-file",
		"token-stdin", "token-env",
		"base-url", "config", "profile",
	}

	for _, name := range expectedFlags {
		if loginCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s to exist on auth login", name)
		}
	}
}

// --- Phase 6: End-to-end cookie auth integration tests ---

// Step 6.1 (end-to-end): Full cookie auth flow through auth test command.
// Mock server at /api/v1/users/self returns 200 with user JSON when Cookie
// header is present (and no Authorization). Verify the CLI command succeeds.
func TestAuthTest_CookieAuth_EndToEnd_FullFlow(t *testing.T) {
	var gotCookie, gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		gotAuth = r.Header.Get("Authorization")

		if r.URL.Path != "/api/v1/users/self" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if gotCookie == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"42","name":"Test Student","login_id":"student@school.edu"}`))
	}))
	defer srv.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: srv.URL,
		Token:   "",
		Cookie:  "canvas_session=end-to-end-test",
		Profile: "default",
	}

	var buf bytes.Buffer
	testCmd := newAuthTestCmd()
	testCmd.SetContext(WithConfig(context.Background(), cfg))
	testCmd.SetOut(&buf)

	err := testCmd.RunE(testCmd, nil)
	if err != nil {
		t.Fatalf("auth test with cookie failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Test Student") {
		t.Errorf("expected user name in output, got: %s", output)
	}
	if !strings.Contains(output, "student@school.edu") {
		t.Errorf("expected login ID in output, got: %s", output)
	}
	if gotCookie != "canvas_session=end-to-end-test" {
		t.Errorf("server received Cookie = %q, want %q", gotCookie, "canvas_session=end-to-end-test")
	}
	if gotAuth != "" {
		t.Errorf("server received Authorization = %q, want empty (cookie auth, no token)", gotAuth)
	}
}

// Step 6.2 (end-to-end): Full cookie auth expiry flow through auth test command.
// Mock server returns 401 when cookie is invalid. Verify the CLI command returns
// an error indicating session expiry.
func TestAuthTest_CookieAuth_EndToEnd_ExpiryFlow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"errors":[{"message":"Invalid access token"}]}`))
	}))
	defer srv.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: srv.URL,
		Token:   "",
		Cookie:  "canvas_session=expired-cookie",
		Profile: "default",
	}

	var buf bytes.Buffer
	testCmd := newAuthTestCmd()
	testCmd.SetContext(WithConfig(context.Background(), cfg))
	testCmd.SetOut(&buf)

	err := testCmd.RunE(testCmd, nil)
	if err == nil {
		t.Fatal("expected error for expired cookie session")
	}
	if !strings.Contains(err.Error(), "session expired") {
		t.Errorf("error = %q, want it to contain 'session expired'", err.Error())
	}
}

// Step 6.2 (end-to-end JSON): Cookie auth expiry in JSON mode returns
// a CANVAS_SESSION_EXPIRED error code.
func TestAuthTest_CookieAuth_EndToEnd_ExpiryFlow_JSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"errors":[{"message":"Invalid access token"}]}`))
	}))
	defer srv.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: srv.URL,
		Token:   "",
		Cookie:  "canvas_session=expired-cookie",
		Profile: "default",
	}

	var buf bytes.Buffer
	testCmd := newAuthTestCmd()
	testCmd.SetContext(WithConfig(context.Background(), cfg))
	testCmd.SetOut(&buf)
	_ = testCmd.Flags().Set("json", "true")

	err := testCmd.RunE(testCmd, nil)
	if err != nil {
		t.Fatalf("JSON mode should return nil error (error is in the envelope): %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if env.OK {
		t.Error("expected ok:false for session expired")
	}
	if env.Error == nil {
		t.Fatal("expected error in envelope")
	}
	if env.Error.Code != "CANVAS_SESSION_EXPIRED" {
		t.Errorf("error code = %q, want %q", env.Error.Code, "CANVAS_SESSION_EXPIRED")
	}
	if !strings.Contains(env.Error.Message, "session expired") {
		t.Errorf("error message = %q, want it to contain 'session expired'", env.Error.Message)
	}
}

// Step 6.3 (end-to-end): CSRF token is cached from response headers and used
// for subsequent mutations through the CLI flow.
func TestAuthTest_CookieAuth_EndToEnd_CSRFCached(t *testing.T) {
	var postCSRF string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Return CSRF token in response header
			w.Header().Set("X-CSRF-Token", "cached-from-response")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"42","name":"CSRF User"}`))
		case http.MethodPost:
			postCSRF = r.Header.Get("X-CSRF-Token")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()

	// Create client with cookie auth, no CSRF token initially.
	c := canvas.NewClient(srv.URL, "", "dev", 5*time.Second, 0).WithCookie("canvas_session=abc", "")

	// GET caches the CSRF token from the response header.
	getResp, err := c.Do(context.Background(), http.MethodGet, "/api/v1/users/self", nil, nil)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	getResp.Body.Close()

	// POST should use the cached CSRF token.
	postResp, err := c.Do(context.Background(), http.MethodPost, "/api/v1/courses/1/assignments", nil, strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("POST error: %v", err)
	}
	postResp.Body.Close()

	if postCSRF != "cached-from-response" {
		t.Errorf("POST X-CSRF-Token = %q, want %q (should be cached from GET response)", postCSRF, "cached-from-response")
	}
}

// Step 6.3 (end-to-end): POST without any CSRF source fails before hitting server.
func TestAuthTest_CookieAuth_EndToEnd_CSRF_Missing(t *testing.T) {
	var serverHit bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverHit = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := canvas.NewClient(srv.URL, "", "dev", 5*time.Second, 0).WithCookie("canvas_session=abc", "")

	_, err := c.Do(context.Background(), http.MethodPost, "/api/v1/courses", nil, strings.NewReader(`{}`))
	if err == nil {
		t.Fatal("expected error for missing CSRF token on POST")
	}
	if serverHit {
		t.Error("server should not be hit when CSRF token is missing")
	}
	if !strings.Contains(err.Error(), "csrf") {
		t.Errorf("error = %q, want it to contain 'csrf'", err.Error())
	}
}

// Step 6.4 (end-to-end): Auth test with cookie receives redirect to /login.
// Verify the CLI returns a session expired error.
func TestAuthTest_CookieAuth_EndToEnd_RedirectToLogin(t *testing.T) {
	var srvURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", srvURL+"/login")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()
	srvURL = srv.URL

	cfg := &config.ResolvedConfig{
		BaseURL: srv.URL,
		Token:   "",
		Cookie:  "canvas_session=redirected",
		Profile: "default",
	}

	var buf bytes.Buffer
	testCmd := newAuthTestCmd()
	testCmd.SetContext(WithConfig(context.Background(), cfg))
	testCmd.SetOut(&buf)

	err := testCmd.RunE(testCmd, nil)
	if err == nil {
		t.Fatal("expected error for redirect to login")
	}
	if !strings.Contains(err.Error(), "session expired") {
		t.Errorf("error = %q, want it to contain 'session expired'", err.Error())
	}
}

// Step 6.5 (end-to-end): Token takes precedence over cookie through auth test.
// Create ResolvedConfig with both token AND cookie. Verify the server receives
// Authorization header (not Cookie).
func TestAuthTest_TokenAndCookie_EndToEnd_TokenPrecedence(t *testing.T) {
	var gotAuth, gotCookie string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotCookie = r.Header.Get("Cookie")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"1","name":"Precedence User"}`))
	}))
	defer srv.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: srv.URL,
		Token:   "my-api-token",
		Cookie:  "canvas_session=should-not-be-used",
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

	if gotAuth != "Bearer my-api-token" {
		t.Errorf("Authorization = %q, want %q (token should take precedence)", gotAuth, "Bearer my-api-token")
	}
	if gotCookie != "" {
		t.Errorf("Cookie = %q, want empty (token takes precedence)", gotCookie)
	}
}
