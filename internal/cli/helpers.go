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

// getClientFromContext retrieves the config from context and creates a canvas client.
// Returns an error if no config is loaded.
func getClientFromContext(ctx context.Context) (*canvas.Client, error) {
	cfg := GetConfig(ctx)
	if cfg == nil {
		return nil, fmt.Errorf("no config loaded")
	}
	return newClientFromCfg(cfg), nil
}

// newClientFromCfg creates a canvas client from a resolved config, applying
// cookie auth when token is absent.
func newClientFromCfg(cfg *config.ResolvedConfig) *canvas.Client {
	client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
	if cfg.Token == "" && cfg.Cookie != "" {
		client.WithCookie(cfg.Cookie, cfg.CSRFToken)
	}
	return client
}

// cookieAuthBaseURL returns cfg.BaseURL as a variadic string slice when cookie
// auth is active (token absent, cookie present). Returns nil otherwise.
// Use this to pass baseURL to NormalizeError/NormalizeErrorFromBody only when
// cookie session expiry detection should apply.
func cookieAuthBaseURL(cfg *config.ResolvedConfig) []string {
	if cfg.Token == "" && cfg.Cookie != "" {
		return []string{cfg.BaseURL}
	}
	return nil
}

// isJSONMode checks the --json flag on the command.
func isJSONMode(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("json")
	return v
}

// writeOutput writes data as a JSON envelope when jsonMode is true,
// or calls humanFn for human-readable output when jsonMode is false.
// If humanFn is nil and jsonMode is false, no output is written.
func writeOutput(w io.Writer, cfg *config.ResolvedConfig, data any, command string, jsonMode bool, humanFn ...func(io.Writer) error) error {
	if jsonMode {
		env := output.NewSuccess(data, command, canvas.Meta{
			Profile: cfg.Profile,
			BaseURL: cfg.BaseURL,
		})
		return output.WriteJSON(w, env, false)
	}
	if len(humanFn) > 0 && humanFn[0] != nil {
		return humanFn[0](w)
	}
	return nil
}

// writeError writes an error as a JSON envelope when jsonMode is true,
// or returns the raw error when jsonMode is false.
func writeError(w io.Writer, err error, command string, jsonMode bool) error {
	if jsonMode {
		env := output.NewError(canvas.ErrorInfo{
			Code:     "CANVAS_API_ERROR",
			Message:  err.Error(),
			Category: "api",
		}, command)
		return output.WriteJSON(w, env, false)
	}
	return err
}

// exitError is an error that carries a process exit code.
type exitError struct {
	msg      string
	exitCode int
}

func (e *exitError) Error() string { return e.msg }
func (e *exitError) ExitCode() int { return e.exitCode }

// checkSafety evaluates the safety policy for a write operation.
// It returns nil if the operation is allowed, or an *exitError if blocked.
func checkSafety(cfg *config.ResolvedConfig, dryRun, confirm bool) error {
	policy := safety.NewPolicy(cfg.ReadOnly, dryRun, confirm, false)
	if err := policy.Check(safety.LowRiskWrite); err != nil {
		var se *safety.SafetyError
		if errors.As(err, &se) {
			return &exitError{msg: se.Message, exitCode: se.ExitCode}
		}
		return err
	}
	return nil
}

// checkHighRiskSafety evaluates the safety policy for a high-risk write operation.
// It returns nil if the operation is allowed, or an *exitError if blocked.
func checkHighRiskSafety(cfg *config.ResolvedConfig, dryRun, confirm bool) error {
	policy := safety.NewPolicy(cfg.ReadOnly, dryRun, confirm, false)
	if err := policy.Check(safety.HighRiskWrite); err != nil {
		var se *safety.SafetyError
		if errors.As(err, &se) {
			return &exitError{msg: se.Message, exitCode: se.ExitCode}
		}
		return err
	}
	return nil
}

// writeAudit writes an audit event for a mutation command.
func writeAudit(cfg *config.ResolvedConfig, command, method, path, body string, dryRun bool) {
	if !cfg.AuditEnabled {
		return
	}
	auditor := audit.NewAuditor(cfg.AuditPath, cfg.AuditEnabled)
	auditor.WriteEvent(canvas.AuditEvent{
		Time:           time.Now().UTC().Format(time.RFC3339),
		SchemaVersion:  output.SchemaVersion,
		Command:        command,
		Profile:        cfg.Profile,
		BaseURL:        cfg.BaseURL,
		Method:         method,
		Path:           path,
		Resource:       map[string]string{},
		RequestHash:    audit.HashBody(body),
		ResponseStatus: 200,
		DryRun:         dryRun,
		Success:        true,
	})
}

// truncateString truncates s to maxLen characters, appending "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
