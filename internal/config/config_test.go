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
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
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

// --- XDG path ---

func TestXDGConfigPath_UsesXDGConfigHome(t *testing.T) {
	dir := t.TempDir()
	setEnv(t, "XDG_CONFIG_HOME", dir)

	got := XDGConfigPath()
	want := filepath.Join(dir, "canvas-cli", "config.yaml")
	if got != want {
		t.Errorf("XDGConfigPath = %q, want %q", got, want)
	}
}

func TestXDGConfigPath_FallsBackToDotConfig(t *testing.T) {
	unsetEnv(t, "XDG_CONFIG_HOME")

	got := XDGConfigPath()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "canvas-cli", "config.yaml")
	if got != want {
		t.Errorf("XDGConfigPath = %q, want %q", got, want)
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

	_, err := Resolve(Options{}, cfg)
	if err == nil {
		t.Fatal("expected error for missing token")
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

// --- homeDir helper on non-Windows ---

func TestHomeDir_ReturnsHomeOrEmpty(t *testing.T) {
	// This just exercises the helper for coverage.
	_ = homeDir()
}

// Ensure the test file compiles on all platforms.
func init() {
	_ = runtime.GOOS
}
