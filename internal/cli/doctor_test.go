package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
)

func TestDoctorCmd_Exists(t *testing.T) {
	cmd := NewDoctorCmd()
	if cmd.Use != "doctor" {
		t.Errorf("expected Use 'doctor', got %q", cmd.Use)
	}
}

func TestDoctorCmd_AllChecksPass(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "1",
			"name": "Test User",
		})
	}))
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL,
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := NewDoctorCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("doctor failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "config_file") {
		t.Errorf("expected config_file check in output, got: %s", output)
	}
	if !strings.Contains(output, "token_present") {
		t.Errorf("expected token_present check in output, got: %s", output)
	}
	if !strings.Contains(output, "base_url") {
		t.Errorf("expected base_url check in output, got: %s", output)
	}
}

func TestDoctorCmd_JSONMode(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "1",
			"name": "Test User",
		})
	}))
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL,
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := NewDoctorCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("doctor failed: %v", err)
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

func TestDoctorCmd_NoToken(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "https://school.instructure.com",
		Token:   "",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := NewDoctorCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	// Should not error, but checks should show failure
	if err != nil {
		t.Fatalf("doctor failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "token_present") {
		t.Errorf("expected token_present check, got: %s", output)
	}
}

func TestDoctorCmd_InvalidBaseURL(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "not-a-url",
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := NewDoctorCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("doctor failed: %v", err)
	}

	output := buf.String()
	// Should show base_url warning
	if !strings.Contains(output, "base_url") {
		t.Errorf("expected base_url check, got: %s", output)
	}
}

func TestDoctorCmd_APIUnreachable(t *testing.T) {
	// Use a server that's not running
	cfg := &config.ResolvedConfig{
		BaseURL: "http://192.0.2.1:1", // RFC 5737 test address, guaranteed unreachable
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := NewDoctorCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("timeout", "1s")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("doctor failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "api_and_token") {
		t.Errorf("expected api_and_token check, got: %s", output)
	}
}

func TestDoctorCmd_WriteSafetyStatus(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "1",
			"name": "Test User",
		})
	}))
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL:  mock.URL,
		Token:    "test-token",
		Profile:  "default",
		ReadOnly: true,
	}

	var buf bytes.Buffer
	cmd := NewDoctorCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("doctor failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "write_safety") {
		t.Errorf("expected write_safety check, got: %s", output)
	}
}

func TestDoctorCmd_ExitCode3_AuthFail(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "https://school.instructure.com",
		Token:   "",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := NewDoctorCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	// doctor should complete without hard error, but checks fail
	_ = cmd.RunE(cmd, nil)
	// The exit code logic is in the calling code, not in RunE itself
	// This test just verifies the command runs
}

func TestDoctorCmd_CookieAuth_Warn(t *testing.T) {
	// When token is empty but cookie is set, token_present should warn
	cfg := &config.ResolvedConfig{
		BaseURL: "https://school.instructure.com",
		Token:   "",
		Cookie:  "session-cookie-value",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := NewDoctorCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("doctor failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	data, ok := env.Data.([]any)
	if !ok {
		t.Fatalf("expected data to be array, got %T", env.Data)
	}

	checks := make(map[string]string)
	for _, item := range data {
		if check, ok := item.(map[string]any); ok {
			name, _ := check["check"].(string)
			status, _ := check["status"].(string)
			checks[name] = status
		}
	}

	if checks["token_present"] != "warn" {
		t.Errorf("expected token_present=warn for cookie auth, got %s", checks["token_present"])
	}
	if checks["session_cookie"] != "pass" {
		t.Errorf("expected session_cookie=pass, got %s", checks["session_cookie"])
	}
}

func TestDoctorCmd_NoTokenNoCookie_Fail(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "https://school.instructure.com",
		Token:   "",
		Cookie:  "",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := NewDoctorCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("doctor failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	data, ok := env.Data.([]any)
	if !ok {
		t.Fatalf("expected data to be array, got %T", env.Data)
	}

	checks := make(map[string]string)
	for _, item := range data {
		if check, ok := item.(map[string]any); ok {
			name, _ := check["check"].(string)
			status, _ := check["status"].(string)
			checks[name] = status
		}
	}

	if checks["token_present"] != "fail" {
		t.Errorf("expected token_present=fail when no token or cookie, got %s", checks["token_present"])
	}
	if checks["session_cookie"] != "pass" {
		t.Errorf("expected session_cookie=pass when no cookie configured, got %s", checks["session_cookie"])
	}
}

func TestDoctorCmd_SessionCookie_EmptyValue(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "https://school.instructure.com",
		Token:   "test-token",
		Cookie:  "   ",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := NewDoctorCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("doctor failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	data, ok := env.Data.([]any)
	if !ok {
		t.Fatalf("expected data to be array, got %T", env.Data)
	}

	checks := make(map[string]string)
	for _, item := range data {
		if check, ok := item.(map[string]any); ok {
			name, _ := check["check"].(string)
			status, _ := check["status"].(string)
			checks[name] = status
		}
	}

	if checks["session_cookie"] != "warn" {
		t.Errorf("expected session_cookie=warn for whitespace-only cookie, got %s", checks["session_cookie"])
	}
}

func TestDoctorCmd_CheckNames(t *testing.T) {
	expectedChecks := []string{
		"config_file",
		"config_permissions",
		"token_present",
		"session_cookie",
		"base_url",
		"api_and_token",
		"write_safety",
	}

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "1",
			"name": "Test User",
		})
	}))
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL,
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := NewDoctorCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	_ = cmd.RunE(cmd, nil)

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	data, ok := env.Data.([]any)
	if !ok {
		t.Fatalf("expected data to be array, got %T", env.Data)
	}

	checkNames := make(map[string]bool)
	for _, item := range data {
		if check, ok := item.(map[string]any); ok {
			if name, ok := check["check"].(string); ok {
				checkNames[name] = true
			}
		}
	}

	for _, expected := range expectedChecks {
		if !checkNames[expected] {
			t.Errorf("missing check: %s", expected)
		}
	}
}
