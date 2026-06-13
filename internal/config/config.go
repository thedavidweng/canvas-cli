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
	Profile string
}

// ResolvedConfig is the final, fully-resolved configuration for a command run.
type ResolvedConfig struct {
	BaseURL          string
	Token            string
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
}

// String returns a human-readable summary with the token redacted.
func (r *ResolvedConfig) String() string {
	return fmt.Sprintf(
		"Profile: %s\nBaseURL: %s\nToken: ***REDACTED***\nTimeout: %s\nRetries: %d\nPageSize: %d\nReadOnly: %t\nDefaultCourse: %s",
		r.Profile, r.BaseURL, r.Timeout, r.Retries, r.PageSize, r.ReadOnly, r.DefaultCourse,
	)
}

// LoadConfig reads and parses a YAML config file. If configPath is empty it
// uses ConfigPath(). If profileOverride is non-empty it overrides
// CurrentProfile.
func LoadConfig(configPath, profileOverride string) (*canvas.Config, error) {
	if configPath == "" {
		configPath = ConfigPath()
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
		if opts.BaseURL == "" && envBase == "" && opts.Token == "" && envToken == "" {
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
	if token == "" {
		return nil, fmt.Errorf("token is required (flag, CANVAS_TOKEN env, or config file)")
	}

	// --- Resolve optional settings (env > file) ---
	timeout := choose("", os.Getenv("CANVAS_TIMEOUT"), prof.Timeout)

	retries := prof.Retries
	if envRetries := os.Getenv("CANVAS_RETRIES"); envRetries != "" {
		if v, err := strconv.Atoi(envRetries); err == nil {
			retries = v
		}
	}

	pageSize := prof.PageSize
	if envPageSize := os.Getenv("CANVAS_PAGE_SIZE"); envPageSize != "" {
		if v, err := strconv.Atoi(envPageSize); err == nil {
			pageSize = v
		}
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
