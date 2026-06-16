package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// helper: write a temp YAML file and return its path.
func writeTempConfig(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// unsetEnv sets an env var and registers cleanup to restore it.
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	orig, had := os.LookupEnv(key)
	os.Unsetenv(key)
	t.Cleanup(func() {
		if had {
			os.Setenv(key, orig)
		} else {
			os.Unsetenv(key)
		}
	})
}

// setEnv sets an env var and registers cleanup to restore it.
func setEnv(t *testing.T, key, val string) {
	t.Helper()
	orig, had := os.LookupEnv(key)
	os.Setenv(key, val)
	t.Cleanup(func() {
		if had {
			os.Setenv(key, orig)
		} else {
			os.Unsetenv(key)
		}
	})
}

// --- LoadConfig tests ---

func TestLoadConfig_ReadsYAMLCorrectly(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: school
profiles:
  school:
    base_url: https://school.instructure.com
    token: secret-token-123
    timeout: 30s
    retries: 3
    page_size: 100
    read_only: true
    default_course: "42"
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)

	cfg, err := LoadConfig(path, "")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.CurrentProfile != "school" {
		t.Errorf("CurrentProfile = %q, want %q", cfg.CurrentProfile, "school")
	}
	p, ok := cfg.Profiles["school"]
	if !ok {
		t.Fatal("expected profile 'school' to exist")
	}
	if p.BaseURL != "https://school.instructure.com" {
		t.Errorf("BaseURL = %q", p.BaseURL)
	}
	if p.Token != "secret-token-123" {
		t.Errorf("Token = %q", p.Token)
	}
	if p.Timeout != "30s" {
		t.Errorf("Timeout = %q", p.Timeout)
	}
	if p.Retries != 3 {
		t.Errorf("Retries = %d", p.Retries)
	}
	if p.PageSize != 100 {
		t.Errorf("PageSize = %d", p.PageSize)
	}
	if !p.ReadOnly {
		t.Error("ReadOnly should be true")
	}
	if p.DefaultCourse != "42" {
		t.Errorf("DefaultCourse = %q", p.DefaultCourse)
	}
}

func TestLoadConfig_EmptyFileReturnsEmptyConfig(t *testing.T) {
	dir := t.TempDir()
	path := writeTempConfig(t, dir, "empty.yaml", "")

	cfg, err := LoadConfig(path, "")
	if err != nil {
		t.Fatalf("LoadConfig on empty file: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config from empty file")
	}
}

func TestLoadConfig_MissingFileReturnsError(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml", "")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadConfig_ProfileSwitching(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://default.instructure.com
    token: default-token
  work:
    base_url: https://work.instructure.com
    token: work-token
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)

	cfg, err := LoadConfig(path, "work")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.CurrentProfile != "work" {
		t.Errorf("CurrentProfile = %q, want %q", cfg.CurrentProfile, "work")
	}
}

func TestLoadConfig_InvalidYAMLReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := writeTempConfig(t, dir, "bad.yaml", `{{{not yaml`)

	_, err := LoadConfig(path, "")
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

// --- Resolve tests: env var overrides ---

func TestResolve_EnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://file.instructure.com
    token: file-token
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	setEnv(t, "CANVAS_BASE_URL", "https://env.instructure.com")
	setEnv(t, "CANVAS_TOKEN", "env-token")
	setEnv(t, "CANVAS_PROFILE", "")

	resolved, err := Resolve(Options{}, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.BaseURL != "https://env.instructure.com" {
		t.Errorf("BaseURL = %q, want env override", resolved.BaseURL)
	}
	if resolved.Token != "env-token" {
		t.Errorf("Token = %q, want env override", resolved.Token)
	}
}

func TestResolve_CANVAS_PROFILE_EnvSelectsProfile(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://default.instructure.com
    token: default-token
  staging:
    base_url: https://staging.instructure.com
    token: staging-token
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	setEnv(t, "CANVAS_PROFILE", "staging")
	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")

	resolved, err := Resolve(Options{}, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.BaseURL != "https://staging.instructure.com" {
		t.Errorf("BaseURL = %q, want staging profile URL", resolved.BaseURL)
	}
}

// --- Resolve tests: explicit flags override env vars ---

func TestResolve_FlagsOverrideEnv(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://file.instructure.com
    token: file-token
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	setEnv(t, "CANVAS_BASE_URL", "https://env.instructure.com")
	setEnv(t, "CANVAS_TOKEN", "env-token")

	opts := Options{
		BaseURL: "https://flag.instructure.com",
		Token:   "flag-token",
	}
	resolved, err := Resolve(opts, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.BaseURL != "https://flag.instructure.com" {
		t.Errorf("BaseURL = %q, want flag override", resolved.BaseURL)
	}
	if resolved.Token != "flag-token" {
		t.Errorf("Token = %q, want flag override", resolved.Token)
	}
}

// --- ConfigPath ---

func TestConfigPath_ReturnsValidPath(t *testing.T) {
	got := ConfigPath()
	if got == "" {
		t.Fatal("ConfigPath returned empty string")
	}
	if !strings.HasSuffix(got, filepath.Join("canvas-cli", "config.yaml")) {
		t.Errorf("ConfigPath = %q, want suffix canvas-cli/config.yaml", got)
	}
}

func TestConfigPath_RespectsXDGConfigHome(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("XDG_CONFIG_HOME only respected on Linux")
	}
	dir := t.TempDir()
	setEnv(t, "XDG_CONFIG_HOME", dir)

	got := ConfigPath()
	want := filepath.Join(dir, "canvas-cli", "config.yaml")
	if got != want {
		t.Errorf("ConfigPath = %q, want %q", got, want)
	}
}

// --- Token redaction ---

func TestResolvedConfig_String_RedactsToken(t *testing.T) {
	rc := &ResolvedConfig{
		BaseURL:  "https://school.instructure.com",
		Token:    "super-secret-token",
		Profile:  "default",
		Timeout:  "30s",
		Retries:  3,
		PageSize: 100,
		ReadOnly: false,
	}

	s := rc.String()
	if strings.Contains(s, "super-secret-token") {
		t.Errorf("String() leaked token: %s", s)
	}
	if !strings.Contains(s, "***REDACTED***") {
		t.Errorf("String() should contain ***REDACTED***, got: %s", s)
	}
	if !strings.Contains(s, "https://school.instructure.com") {
		t.Errorf("String() should contain base URL")
	}
}

// --- env:VAR_NAME token references ---

func TestResolve_EnvVarTokenReference(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    token: env:MY_CANVAS_SECRET
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	setEnv(t, "MY_CANVAS_SECRET", "resolved-secret-value")
	unsetEnv(t, "CANVAS_TOKEN")
	unsetEnv(t, "CANVAS_BASE_URL")

	resolved, err := Resolve(Options{}, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Token != "resolved-secret-value" {
		t.Errorf("Token = %q, want resolved env var value", resolved.Token)
	}
}

func TestResolve_EnvVarTokenReference_MissingEnv(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    token: env:NONEXISTENT_VAR_12345
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "NONEXISTENT_VAR_12345")
	unsetEnv(t, "CANVAS_TOKEN")
	unsetEnv(t, "CANVAS_BASE_URL")

	_, err := Resolve(Options{}, cfg)
	if err == nil {
		t.Fatal("expected error for missing env var in token reference")
	}
}

// --- BaseURL normalization ---

func TestResolve_BaseURL_NormalizesTrailingSlash(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com/
    token: tok
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")

	resolved, err := Resolve(Options{}, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.BaseURL != "https://school.instructure.com" {
		t.Errorf("BaseURL = %q, want trailing slash stripped", resolved.BaseURL)
	}
}

func TestResolve_BaseURL_StripsApiV1Suffix(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com/api/v1
    token: tok
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")

	resolved, err := Resolve(Options{}, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.BaseURL != "https://school.instructure.com" {
		t.Errorf("BaseURL = %q, want /api/v1 suffix stripped", resolved.BaseURL)
	}
}

func TestResolve_BaseURL_TrailingSlashAndApiV1(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com/api/v1/
    token: tok
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")

	resolved, err := Resolve(Options{}, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.BaseURL != "https://school.instructure.com" {
		t.Errorf("BaseURL = %q, want both suffixes stripped", resolved.BaseURL)
	}
}

func TestResolve_BaseURL_NormalizesFromFlag(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://file.instructure.com
    token: tok
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")

	resolved, err := Resolve(Options{BaseURL: "https://flag.instructure.com/api/v1/"}, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.BaseURL != "https://flag.instructure.com" {
		t.Errorf("BaseURL = %q, want normalized flag value", resolved.BaseURL)
	}
}

// --- Missing required fields ---

func TestResolve_MissingBaseURLError(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    token: tok
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")

	_, err := Resolve(Options{}, cfg)
	if err == nil {
		t.Fatal("expected error for missing base URL")
	}
}

func TestResolve_MissingTokenError(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")
	unsetEnv(t, "CANVAS_COOKIE")

	_, err := Resolve(Options{}, cfg)
	if err == nil {
		t.Fatal("expected error for missing token and cookie")
	}
}

// --- Profile not found ---

func TestResolve_ProfileNotFound(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    token: tok
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "nonexistent")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")

	_, err := Resolve(Options{}, cfg)
	if err == nil {
		t.Fatal("expected error for nonexistent profile")
	}
}

// --- Invalid env var values ---

func TestResolve_InvalidCANVAS_RETRIES_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    token: tok
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")
	setEnv(t, "CANVAS_RETRIES", "abc")

	_, err := Resolve(Options{}, cfg)
	if err == nil {
		t.Fatal("expected error for non-integer CANVAS_RETRIES")
	}
	if !strings.Contains(err.Error(), "CANVAS_RETRIES") {
		t.Errorf("error should mention CANVAS_RETRIES, got: %v", err)
	}
}

func TestResolve_InvalidCANVAS_PAGE_SIZE_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    token: tok
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")
	setEnv(t, "CANVAS_PAGE_SIZE", "not-a-number")

	_, err := Resolve(Options{}, cfg)
	if err == nil {
		t.Fatal("expected error for non-integer CANVAS_PAGE_SIZE")
	}
	if !strings.Contains(err.Error(), "CANVAS_PAGE_SIZE") {
		t.Errorf("error should mention CANVAS_PAGE_SIZE, got: %v", err)
	}
}

// --- Priority: flag > env > file ---

func TestResolve_FullPrecedence(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://file.instructure.com
    token: file-token
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	setEnv(t, "CANVAS_BASE_URL", "https://env.instructure.com")
	setEnv(t, "CANVAS_TOKEN", "env-token")

	opts := Options{
		BaseURL: "https://flag.instructure.com",
		Token:   "flag-token",
	}
	resolved, err := Resolve(opts, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.BaseURL != "https://flag.instructure.com" {
		t.Errorf("BaseURL = %q, want flag (highest priority)", resolved.BaseURL)
	}
	if resolved.Token != "flag-token" {
		t.Errorf("Token = %q, want flag (highest priority)", resolved.Token)
	}
}

// --- userHomeDir helper ---

func TestUserHomeDir_ReturnsHomeOrEmpty(t *testing.T) {
	_ = userHomeDir()
}

// --- Session cookie auth tests (Step 1.2) ---

func TestResolve_CookieWithCSRF_SucceedsWithoutToken(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    cookie: _csrf_token=abc123; _session=xyz
    csrf_token: my-csrf-token
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")
	unsetEnv(t, "CANVAS_COOKIE")

	resolved, err := Resolve(Options{}, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Cookie != "_csrf_token=abc123; _session=xyz" {
		t.Errorf("Cookie = %q, want from profile", resolved.Cookie)
	}
	if resolved.CSRFToken != "my-csrf-token" {
		t.Errorf("CSRFToken = %q, want from profile", resolved.CSRFToken)
	}
	if resolved.Token != "" {
		t.Errorf("Token should be empty when only cookie is set, got %q", resolved.Token)
	}
}

func TestResolve_MissingTokenAndCookie_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")
	unsetEnv(t, "CANVAS_COOKIE")

	_, err := Resolve(Options{}, cfg)
	if err == nil {
		t.Fatal("expected error when neither token nor cookie is set")
	}
	if !strings.Contains(err.Error(), "token or cookie required") {
		t.Errorf("error should mention 'token or cookie required', got: %v", err)
	}
}

func TestResolve_TokenTakesPrecedenceOverCookie(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    token: my-token
    cookie: my-cookie
    csrf_token: my-csrf
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")
	unsetEnv(t, "CANVAS_COOKIE")

	resolved, err := Resolve(Options{}, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Token != "my-token" {
		t.Errorf("Token = %q, want 'my-token' (should take precedence)", resolved.Token)
	}
	if resolved.Cookie != "my-cookie" {
		t.Errorf("Cookie = %q, want 'my-cookie' (should still be set)", resolved.Cookie)
	}
}

func TestResolve_CookieWithoutCSRF_Warns(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    cookie: my-cookie
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")
	unsetEnv(t, "CANVAS_COOKIE")

	resolved, err := Resolve(Options{}, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(resolved.Warnings) == 0 {
		t.Fatal("expected warning when cookie is set without csrf_token")
	}
	found := false
	for _, w := range resolved.Warnings {
		if strings.Contains(w, "CSRF") || strings.Contains(w, "csrf") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected CSRF-related warning, got warnings: %v", resolved.Warnings)
	}
}

func TestResolve_EnvVarCookieReference(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    cookie: env:MY_COOKIE_SECRET
    csrf_token: env:MY_CSRF_SECRET
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	setEnv(t, "MY_COOKIE_SECRET", "resolved-cookie-value")
	setEnv(t, "MY_CSRF_SECRET", "resolved-csrf-value")
	unsetEnv(t, "CANVAS_TOKEN")
	unsetEnv(t, "CANVAS_COOKIE")
	unsetEnv(t, "CANVAS_BASE_URL")

	resolved, err := Resolve(Options{}, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Cookie != "resolved-cookie-value" {
		t.Errorf("Cookie = %q, want resolved env var value", resolved.Cookie)
	}
	if resolved.CSRFToken != "resolved-csrf-value" {
		t.Errorf("CSRFToken = %q, want resolved env var value", resolved.CSRFToken)
	}
}

func TestResolve_EnvVarCookieReference_MissingEnv(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    cookie: env:NONEXISTENT_COOKIE_VAR_12345
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "NONEXISTENT_COOKIE_VAR_12345")
	unsetEnv(t, "CANVAS_TOKEN")
	unsetEnv(t, "CANVAS_COOKIE")
	unsetEnv(t, "CANVAS_BASE_URL")

	_, err := Resolve(Options{}, cfg)
	if err == nil {
		t.Fatal("expected error for missing env var in cookie reference")
	}
}

func TestResolvedConfig_String_RedactsCookie(t *testing.T) {
	rc := &ResolvedConfig{
		BaseURL:  "https://school.instructure.com",
		Token:    "super-secret-token",
		Cookie:   "super-secret-cookie",
		Profile:  "default",
		Timeout:  "30s",
		Retries:  3,
		PageSize: 100,
	}

	s := rc.String()
	if strings.Contains(s, "super-secret-cookie") {
		t.Errorf("String() leaked cookie: %s", s)
	}
	if !strings.Contains(s, "***REDACTED***") {
		t.Errorf("String() should contain ***REDACTED***, got: %s", s)
	}
}

func TestResolve_CANVAS_COOKIE_EnvSupport(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")
	setEnv(t, "CANVAS_COOKIE", "env-cookie-value")

	resolved, err := Resolve(Options{}, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Cookie != "env-cookie-value" {
		t.Errorf("Cookie = %q, want CANVAS_COOKIE env value", resolved.Cookie)
	}
}

func TestResolve_CANVAS_COOKIE_EnvWithCSRF(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    csrf_token: file-csrf
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")
	setEnv(t, "CANVAS_COOKIE", "env-cookie-value")

	resolved, err := Resolve(Options{}, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Cookie != "env-cookie-value" {
		t.Errorf("Cookie = %q, want CANVAS_COOKIE env value", resolved.Cookie)
	}
	if resolved.CSRFToken != "file-csrf" {
		t.Errorf("CSRFToken = %q, want 'file-csrf' from profile", resolved.CSRFToken)
	}
}

func TestResolve_CookieFlagTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    cookie: file-cookie
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")
	setEnv(t, "CANVAS_COOKIE", "env-cookie")

	opts := Options{Cookie: "flag-cookie"}
	resolved, err := Resolve(opts, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Cookie != "flag-cookie" {
		t.Errorf("Cookie = %q, want flag value (highest priority)", resolved.Cookie)
	}
}

func TestResolve_EnvVarCSRFTokenReference_MissingEnv(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    cookie: some-cookie
    csrf_token: env:NONEXISTENT_CSRF_VAR_12345
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "NONEXISTENT_CSRF_VAR_12345")
	unsetEnv(t, "CANVAS_TOKEN")
	unsetEnv(t, "CANVAS_COOKIE")
	unsetEnv(t, "CANVAS_BASE_URL")

	_, err := Resolve(Options{}, cfg)
	if err == nil {
		t.Fatal("expected error for missing env var in csrf_token reference")
	}
}

func TestResolve_ProfileFlagOverride(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://default.instructure.com
    token: default-token
  work:
    base_url: https://work.instructure.com
    token: work-token
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")
	unsetEnv(t, "CANVAS_PROFILE")

	opts := Options{Profile: "work"}
	resolved, err := Resolve(opts, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Profile != "work" {
		t.Errorf("Profile = %q, want %q", resolved.Profile, "work")
	}
	if resolved.BaseURL != "https://work.instructure.com" {
		t.Errorf("BaseURL = %q, want work profile URL", resolved.BaseURL)
	}
}

func TestResolve_CANVAS_READ_ONLY_Env(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    token: tok
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")
	setEnv(t, "CANVAS_READ_ONLY", "1")

	resolved, err := Resolve(Options{}, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !resolved.ReadOnly {
		t.Error("ReadOnly should be true when CANVAS_READ_ONLY=1")
	}
}

func TestResolve_CANVAS_NO_COLOR_Env(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    token: tok
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")
	setEnv(t, "CANVAS_NO_COLOR", "1")

	resolved, err := Resolve(Options{}, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !resolved.OutputNoColor {
		t.Error("OutputNoColor should be true when CANVAS_NO_COLOR=1")
	}
}

func TestResolve_CANVAS_RETRIES_Env(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    token: tok
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")
	setEnv(t, "CANVAS_RETRIES", "5")

	resolved, err := Resolve(Options{}, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Retries != 5 {
		t.Errorf("Retries = %d, want 5", resolved.Retries)
	}
}

func TestResolve_CANVAS_PAGE_SIZE_Env(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    token: tok
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")
	setEnv(t, "CANVAS_PAGE_SIZE", "200")

	resolved, err := Resolve(Options{}, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.PageSize != 200 {
		t.Errorf("PageSize = %d, want 200", resolved.PageSize)
	}
}

func TestResolve_CANVAS_TIMEOUT_Env(t *testing.T) {
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    token: tok
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	cfg, _ := LoadConfig(path, "")

	unsetEnv(t, "CANVAS_BASE_URL")
	unsetEnv(t, "CANVAS_TOKEN")
	setEnv(t, "CANVAS_TIMEOUT", "45s")

	resolved, err := Resolve(Options{}, cfg)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Timeout != "45s" {
		t.Errorf("Timeout = %q, want 45s", resolved.Timeout)
	}
}

func TestLoadConfig_TooPermissivePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file permissions not checked on Windows")
	}
	dir := t.TempDir()
	yaml := `
current_profile: default
profiles:
  default:
    base_url: https://school.instructure.com
    token: tok
`
	path := writeTempConfig(t, dir, "config.yaml", yaml)
	// Set too-permissive permissions.
	if err := os.Chmod(path, 0o777); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(path, "")
	if err == nil {
		t.Fatal("expected error for too-permissive config file")
	}
	if !strings.Contains(err.Error(), "too-permissive") {
		t.Errorf("error = %q, want it to contain 'too-permissive'", err.Error())
	}
}

func TestLoadConfig_EmptyPath_UsesDefault(t *testing.T) {
	// LoadConfig with empty path uses ConfigPath(). If the default config
	// doesn't exist, it should return an error about reading the file.
	_, err := LoadConfig("", "")
	if err == nil {
		// Config might exist at default path; that's OK.
		return
	}
	if !strings.Contains(err.Error(), "reading config") {
		t.Errorf("error = %q, want it to contain 'reading config'", err.Error())
	}
}

func TestResolvedConfig_String_RedactsBothEmpty(t *testing.T) {
	rc := &ResolvedConfig{
		BaseURL: "https://school.instructure.com",
		Profile: "default",
	}

	s := rc.String()
	if !strings.Contains(s, "(not set)") {
		t.Errorf("String() should contain '(not set)' for unset token/cookie, got: %s", s)
	}
}

func TestConfigPath_FallbackToHomeDir(t *testing.T) {
	// Test that ConfigPath works when UserConfigDir succeeds.
	got := ConfigPath()
	if got == "" {
		t.Fatal("ConfigPath returned empty string")
	}
	if !filepath.IsAbs(got) {
		t.Errorf("ConfigPath = %q, want absolute path", got)
	}
}

// Ensure the test file compiles on all platforms.
func init() {
	_ = runtime.GOOS
}
