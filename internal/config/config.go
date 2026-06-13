// Package config loads, merges, and resolves canvas-cli configuration.
//
// Precedence: explicit flags > environment variables > config file > defaults.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
)

// Options holds explicit flag values that override everything else.
type Options struct {
	BaseURL string
	Token   string
	Cookie  string
	Profile string
}

// ResolvedConfig is the final, fully-resolved configuration for a command run.
type ResolvedConfig struct {
	BaseURL          string
	Token            string
	Cookie           string
	CSRFToken        string
	Profile          string
	Timeout          string
	TimeoutDuration  time.Duration
	Retries          int
	PageSize         int
	Limit            int
	NoPaginate       bool
	ReadOnly         bool
	DefaultCourse    string
	OutputJSONPretty bool
	OutputNoColor    bool
	AuditEnabled     bool
	AuditPath        string
	Warnings         []string
}

// String returns a human-readable summary with sensitive values redacted.
func (r *ResolvedConfig) String() string {
	tokenStr := "(not set)"
	if r.Token != "" {
		tokenStr = "***REDACTED***"
	}
	cookieStr := "(not set)"
	if r.Cookie != "" {
		cookieStr = "***REDACTED***"
	}
	return fmt.Sprintf(
		"Profile: %s\nBaseURL: %s\nToken: %s\nCookie: %s\nTimeout: %s\nRetries: %d\nPageSize: %d\nReadOnly: %t\nDefaultCourse: %s",
		r.Profile, r.BaseURL, tokenStr, cookieStr, r.Timeout, r.Retries, r.PageSize, r.ReadOnly, r.DefaultCourse,
	)
}

// LoadConfig reads and parses a YAML config file. If configPath is empty it
// uses ConfigPath(). If profileOverride is non-empty it overrides
// CurrentProfile. Rejects files with permissions more permissive than 0600.
func LoadConfig(configPath, profileOverride string) (*canvas.Config, error) {
	if configPath == "" {
		configPath = ConfigPath()
	}

	// Reject group/world-readable config files (security requirement).
	if info, err := os.Stat(configPath); err == nil {
		perm := info.Mode().Perm()
		if perm&0o077 != 0 {
			return nil, fmt.Errorf("config file %s has too-permissive permissions (%o); must be 0600 or stricter", configPath, perm)
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", configPath, err)
	}

	cfg := &canvas.Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", configPath, err)
	}

	if profileOverride != "" {
		cfg.CurrentProfile = profileOverride
	}

	return cfg, nil
}

// Resolve merges Options (flags), environment variables, and the parsed config
// into a ResolvedConfig. It returns an error when required values are missing.
func Resolve(opts Options, cfg *canvas.Config) (*ResolvedConfig, error) {
	// Determine effective profile name.
	profileName := cfg.CurrentProfile
	if opts.Profile != "" {
		profileName = opts.Profile
	}
	if envProfile := os.Getenv("CANVAS_PROFILE"); envProfile != "" && opts.Profile == "" {
		profileName = envProfile
	}

	// Look up profile. Allow missing profile when env vars or flags provide
	// the required values (env-var-first configuration).
	prof, ok := cfg.Profiles[profileName]
	if !ok && profileName != "" {
		// Check if env vars or flags will provide base URL and token.
		envBase := os.Getenv("CANVAS_BASE_URL")
		envToken := os.Getenv("CANVAS_TOKEN")
		envCookie := os.Getenv("CANVAS_COOKIE")
		if opts.BaseURL == "" && envBase == "" && opts.Token == "" && envToken == "" && opts.Cookie == "" && envCookie == "" {
			return nil, fmt.Errorf("profile %q not found in config", profileName)
		}
		// Profile missing but env/flag credentials are available; proceed with zero-value profile.
		prof = canvas.Profile{}
	}

	// --- Resolve BaseURL (flag > env > file) ---
	baseURL := choose(opts.BaseURL, os.Getenv("CANVAS_BASE_URL"), prof.BaseURL)
	baseURL = normalizeBaseURL(baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("base URL is required (flag, CANVAS_BASE_URL env, or config file)")
	}

	// --- Resolve Token (flag > env > file) ---
	token := choose(opts.Token, os.Getenv("CANVAS_TOKEN"), prof.Token)
	if strings.HasPrefix(token, "env:") {
		envKey := strings.TrimPrefix(token, "env:")
		resolved := os.Getenv(envKey)
		if resolved == "" {
			return nil, fmt.Errorf("token references env var %q which is not set", envKey)
		}
		token = resolved
	}

	// --- Resolve Cookie (flag > env > file) ---
	cookie := choose(opts.Cookie, os.Getenv("CANVAS_COOKIE"), prof.Cookie)
	if strings.HasPrefix(cookie, "env:") {
		envKey := strings.TrimPrefix(cookie, "env:")
		resolved := os.Getenv(envKey)
		if resolved == "" {
			return nil, fmt.Errorf("cookie references env var %q which is not set", envKey)
		}
		cookie = resolved
	}

	// --- Resolve CSRF Token (env > file, same env: pattern) ---
	csrfToken := prof.CSRFToken
	if strings.HasPrefix(csrfToken, "env:") {
		envKey := strings.TrimPrefix(csrfToken, "env:")
		resolved := os.Getenv(envKey)
		if resolved == "" {
			return nil, fmt.Errorf("csrf_token references env var %q which is not set", envKey)
		}
		csrfToken = resolved
	}

	// --- Require at least one auth method ---
	var warnings []string
	if token == "" && cookie == "" {
		return nil, fmt.Errorf("token or cookie required (cookie auth is experimental)")
	}

	// Warn when cookie is set without CSRF token (write commands will fail).
	if cookie != "" && csrfToken == "" && token == "" {
		warnings = append(warnings, "cookie auth without CSRF token: write commands will fail")
	}

	// --- Resolve optional settings (env > file) ---
	timeout := choose("", os.Getenv("CANVAS_TIMEOUT"), prof.Timeout)

	retries := prof.Retries
	if envRetries := os.Getenv("CANVAS_RETRIES"); envRetries != "" {
		v, err := strconv.Atoi(envRetries)
		if err != nil {
			return nil, fmt.Errorf("invalid CANVAS_RETRIES value %q: %w", envRetries, err)
		}
		retries = v
	}

	pageSize := prof.PageSize
	if envPageSize := os.Getenv("CANVAS_PAGE_SIZE"); envPageSize != "" {
		v, err := strconv.Atoi(envPageSize)
		if err != nil {
			return nil, fmt.Errorf("invalid CANVAS_PAGE_SIZE value %q: %w", envPageSize, err)
		}
		pageSize = v
	}

	readOnly := prof.ReadOnly
	if os.Getenv("CANVAS_READ_ONLY") == "1" {
		readOnly = true
	}

	noColor := cfg.Output.NoColor
	if os.Getenv("CANVAS_NO_COLOR") == "1" {
		noColor = true
	}

	return &ResolvedConfig{
		BaseURL:          baseURL,
		Token:            token,
		Cookie:           cookie,
		CSRFToken:        csrfToken,
		Profile:          profileName,
		Timeout:          timeout,
		Retries:          retries,
		PageSize:         pageSize,
		ReadOnly:         readOnly,
		DefaultCourse:    prof.DefaultCourse,
		OutputJSONPretty: cfg.Output.JSONPretty,
		OutputNoColor:    noColor,
		AuditEnabled:     cfg.Audit.Enabled,
		AuditPath:        cfg.Audit.Path,
		Warnings:         warnings,
	}, nil
}

// ConfigPath returns the default config file path using the OS-appropriate
// config directory:
//
//	Linux:   $XDG_CONFIG_HOME/canvas-cli/config.yaml  (default ~/.config)
//	macOS:   ~/Library/Application Support/canvas-cli/config.yaml
//	Windows: %APPDATA%\canvas-cli\config.yaml
func ConfigPath() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base = filepath.Join(userHomeDir(), ".config")
	}
	return filepath.Join(base, "canvas-cli", "config.yaml")
}

// userHomeDir returns the user's home directory. On error it returns "".
func userHomeDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return ""
}

// choose returns the first non-empty string from the arguments, implementing
// the precedence: explicit > env > file.
func choose(explicit, env, file string) string {
	if explicit != "" {
		return explicit
	}
	if env != "" {
		return env
	}
	return file
}

// normalizeBaseURL strips a trailing slash and an /api/v1 suffix so callers
// get a clean root URL like https://school.instructure.com.
func normalizeBaseURL(u string) string {
	u = strings.TrimRight(u, "/")
	u = strings.TrimSuffix(u, "/api/v1")
	return u
}
