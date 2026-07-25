package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
	"github.com/thedavidweng/canvas-cli/internal/output"
	"github.com/thedavidweng/canvas-cli/internal/safety"
)

// MutationSpec describes a write operation for the centralized safety,
// dry-run, audit, and error-normalization pipeline.
type MutationSpec struct {
	Command        string
	Level          safety.SafetyLevel
	Method         string
	Path           string
	DryRun         bool
	Confirm        bool
	ResourceIDs    []string
	PayloadSummary string
	AuditBody      string
	Resource       map[string]string
}

// Doer performs the Canvas API call inside Run, returning the result data (for
// JSON output), the HTTP response status (for audit), and any error.
type Doer func(ctx context.Context, client *canvas.Client) (data any, responseStatus int, err error)

// CheckAndPreview evaluates the safety policy and, on dry-run, prints the
// preview to w and returns true so the handler short-circuits. Returns an
// *exitError when the safety policy blocks the operation.
func CheckAndPreview(cfg *config.ResolvedConfig, w io.Writer, spec *MutationSpec) (bool, error) {
	if err := checkSafetyLevel(cfg, spec.DryRun, spec.Confirm, spec.Level); err != nil {
		return false, err
	}
	if spec.DryRun {
		preview := safety.FormatPreview(safety.Preview{
			Method:         spec.Method,
			Path:           spec.Path,
			ResourceIDs:    spec.ResourceIDs,
			PayloadSummary: spec.PayloadSummary,
		})
		fmt.Fprintln(w, preview)
		return true, nil
	}
	return false, nil
}

// RecordAudit writes an audit event for a mutation outcome.
func RecordAudit(cfg *config.ResolvedConfig, spec *MutationSpec, responseStatus int, success bool) {
	writeAuditWithResource(cfg, spec.Command, spec.Method, spec.Path, spec.AuditBody, false, responseStatus, success, spec.Resource)
}

// Run executes a mutation through the full pipeline: safety check, dry-run
// preview, canvas call, audit logging, and output. In JSON mode it writes the
// success/error envelope; otherwise humanFn (if non-nil) renders the result.
func Run(ctx context.Context, cfg *config.ResolvedConfig, w io.Writer, jsonMode bool, spec *MutationSpec, do Doer, humanFn func(w io.Writer, data any) error) error {
	dryRun, err := CheckAndPreview(cfg, w, spec)
	if err != nil {
		return err
	}
	if dryRun {
		return nil
	}

	client := newClientFromCfg(cfg)
	data, responseStatus, err := do(ctx, client)
	if err != nil {
		RecordAudit(cfg, spec, responseStatus, false)
		return writeError(w, cfg, err, spec.Command, jsonMode)
	}

	RecordAudit(cfg, spec, responseStatus, true)

	if jsonMode {
		env := output.NewSuccess(data, spec.Command, canvas.Meta{
			Profile: cfg.Profile,
			BaseURL: cfg.BaseURL,
		})
		return writeEnvelope(w, cfg, &env)
	}

	if humanFn != nil {
		return humanFn(w, data)
	}
	return nil
}
