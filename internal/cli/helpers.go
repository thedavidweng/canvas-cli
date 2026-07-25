package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/audit"
	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
	"github.com/thedavidweng/canvas-cli/internal/output"
	"github.com/thedavidweng/canvas-cli/internal/safety"
)

func getClientFromContext(ctx context.Context) (*canvas.Client, error) {
	cfg := GetConfig(ctx)
	if cfg == nil {
		return nil, fmt.Errorf("no config loaded")
	}
	return newClientFromCfg(cfg), nil
}

func newClientFromCfg(cfg *config.ResolvedConfig) *canvas.Client {
	client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
	if cfg.Token == "" && cfg.Cookie != "" {
		client.WithCookie(cfg.Cookie, cfg.CSRFToken)
	}
	return client
}

// cookieAuthBaseURL returns cfg.BaseURL as a variadic slice when cookie auth is
// active (token absent, cookie present), enabling session-expiry detection in
// NormalizeError. Returns nil otherwise.
func cookieAuthBaseURL(cfg *config.ResolvedConfig) []string {
	if cfg.Token == "" && cfg.Cookie != "" {
		return []string{cfg.BaseURL}
	}
	return nil
}

func isJSONMode(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("json")
	return v
}

// writeEnvelope serializes a pre-built envelope, honoring --pretty from cfg.
func writeEnvelope(w io.Writer, cfg *config.ResolvedConfig, env canvas.Envelope) error {
	return output.WriteJSON(w, env, cfg.OutputJSONPretty)
}

// writeOutput writes data as a JSON envelope when jsonMode is true, or calls
// humanFn otherwise. With no humanFn and jsonMode false, nothing is written.
func writeOutput(w io.Writer, cfg *config.ResolvedConfig, data any, command string, jsonMode bool, humanFn ...func(io.Writer) error) error {
	if jsonMode {
		env := output.NewSuccess(data, command, canvas.Meta{
			Profile: cfg.Profile,
			BaseURL: cfg.BaseURL,
		})
		return output.WriteJSON(w, env, cfg.OutputJSONPretty)
	}
	if len(humanFn) > 0 && humanFn[0] != nil {
		return humanFn[0](w)
	}
	return nil
}

func writeError(w io.Writer, cfg *config.ResolvedConfig, err error, command string, jsonMode bool) error {
	return writeErrorWithCode(w, cfg, err, command, "CANVAS_API_ERROR", "api", jsonMode)
}

func writeNetworkError(w io.Writer, cfg *config.ResolvedConfig, err error, command string, jsonMode bool) error {
	return writeErrorWithCode(w, cfg, err, command, "CANVAS_NETWORK_ERROR", "network", jsonMode)
}

// writeErrorWithCode writes an error envelope when jsonMode is true; otherwise
// it returns an *exitError carrying the process exit code mapped from category.
func writeErrorWithCode(w io.Writer, cfg *config.ResolvedConfig, err error, command, code, category string, jsonMode bool) error {
	if jsonMode {
		env := output.NewError(canvas.ErrorInfo{
			Code:     code,
			Message:  err.Error(),
			Category: category,
		}, command)
		return output.WriteJSON(w, env, cfg.OutputJSONPretty)
	}
	return &exitError{msg: err.Error(), exitCode: output.ExitCodeForCategory(category)}
}

// exitError is an error that carries a process exit code.
type exitError struct {
	msg      string
	exitCode int
}

func (e *exitError) Error() string { return e.msg }
func (e *exitError) ExitCode() int { return e.exitCode }

// checkSafetyLevel evaluates the safety policy at the given level, returning an
// *exitError carrying the safety exit code when the operation is blocked.
func checkSafetyLevel(cfg *config.ResolvedConfig, dryRun, confirm bool, level safety.SafetyLevel) error {
	policy := safety.NewPolicy(cfg.ReadOnly, dryRun, confirm)
	if err := policy.Check(level); err != nil {
		var se *safety.SafetyError
		if errors.As(err, &se) {
			return &exitError{msg: se.Message, exitCode: se.ExitCode}
		}
		return err
	}
	return nil
}

func writeAudit(cfg *config.ResolvedConfig, command, method, path, body string, dryRun bool, responseStatus int, success bool) {
	writeAuditWithResource(cfg, command, method, path, body, dryRun, responseStatus, success, nil)
}

func writeAuditWithResource(cfg *config.ResolvedConfig, command, method, path, body string, dryRun bool, responseStatus int, success bool, resource map[string]string) {
	if !cfg.AuditEnabled {
		return
	}
	if resource == nil {
		resource = map[string]string{}
	}
	auditor := audit.NewAuditor(cfg.AuditPath, cfg.AuditEnabled)
	_ = auditor.WriteEvent(canvas.AuditEvent{
		Time:           time.Now().UTC().Format(time.RFC3339),
		SchemaVersion:  output.SchemaVersion,
		Command:        command,
		Profile:        cfg.Profile,
		BaseURL:        cfg.BaseURL,
		Method:         method,
		Path:           path,
		Resource:       resource,
		RequestHash:    audit.HashBody(body),
		ResponseStatus: responseStatus,
		DryRun:         dryRun,
		Success:        success,
	})
}

func truncateString(s string, maxLen int) string {
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	return string(r[:maxLen]) + "..."
}
