package safety

import "github.com/thedavidweng/canvas-cli/internal/output"

// SafetyLevel represents the risk level of an operation.
type SafetyLevel int

const (
	ReadLevel SafetyLevel = iota
	LowRiskWrite
	HighRiskWrite
)

func (l SafetyLevel) String() string {
	switch l {
	case ReadLevel:
		return "ReadLevel"
	case LowRiskWrite:
		return "LowRiskWrite"
	case HighRiskWrite:
		return "HighRiskWrite"
	default:
		return "Unknown"
	}
}

// SafetyError is returned when a safety policy blocks an operation.
type SafetyError struct {
	Message  string
	ExitCode int
}

func (e *SafetyError) Error() string { return e.Message }

var (
	ErrSafetyBlocked = &SafetyError{
		Message:  "operation blocked by read-only mode",
		ExitCode: output.CodeSafetyBlocked,
	}

	ErrNeedsConfirm = &SafetyError{
		Message:  "operation requires --confirm",
		ExitCode: output.CodeSafetyBlocked,
	}
)

// Policy holds the safety flags for a command invocation.
type Policy struct {
	ReadOnly bool
	DryRun   bool
	Confirm  bool
}

func NewPolicy(readOnly, dryRun, confirm bool) Policy {
	return Policy{ReadOnly: readOnly, DryRun: dryRun, Confirm: confirm}
}

// Check verifies that the policy allows the operation at the given safety level.
// It returns nil if allowed, or a *SafetyError if blocked. --dry-run is always
// allowed; --read-only blocks all writes even with --confirm.
func (p Policy) Check(level SafetyLevel) error {
	if level == ReadLevel || p.DryRun {
		return nil
	}
	if p.ReadOnly {
		return ErrSafetyBlocked
	}
	if !p.Confirm {
		return ErrNeedsConfirm
	}
	return nil
}
