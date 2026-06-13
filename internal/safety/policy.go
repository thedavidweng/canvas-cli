package safety

import "os"

// SafetyLevel represents the risk level of an operation.
type SafetyLevel int

const (
	ReadLevel     SafetyLevel = 0
	LowRiskWrite  SafetyLevel = 1
	HighRiskWrite SafetyLevel = 2
	Destructive   SafetyLevel = 3
)

// String returns a human-readable name for the safety level.
func (l SafetyLevel) String() string {
	switch l {
	case ReadLevel:
		return "ReadLevel"
	case LowRiskWrite:
		return "LowRiskWrite"
	case HighRiskWrite:
		return "HighRiskWrite"
	case Destructive:
		return "Destructive"
	default:
		return "Unknown"
	}
}

// SafetyError is returned when a safety policy blocks an operation.
type SafetyError struct {
	Message  string
	ExitCode int
}

func (e *SafetyError) Error() string {
	return e.Message
}

// Sentinel errors for safety checks.
var (
	ErrSafetyBlocked = &SafetyError{
		Message:  "operation blocked by read-only mode",
		ExitCode: 7,
	}

	ErrNeedsConfirm = &SafetyError{
		Message:  "operation requires --confirm",
		ExitCode: 0,
	}

	ErrNeedsConfirmDelete = &SafetyError{
		Message:  "operation requires --confirm-delete",
		ExitCode: 0,
	}
)

// Policy holds the safety flags for a command invocation.
type Policy struct {
	ReadOnly      bool
	DryRun        bool
	Confirm       bool
	ConfirmDelete bool
}

// NewPolicy creates a Policy from the given flag values.
func NewPolicy(readOnly, dryRun, confirm, confirmDelete bool) Policy {
	return Policy{
		ReadOnly:      readOnly,
		DryRun:        dryRun,
		Confirm:       confirm,
		ConfirmDelete: confirmDelete,
	}
}

// Check verifies that the policy allows the operation at the given safety level.
// It returns nil if the operation is allowed, or a *SafetyError if blocked.
func (p Policy) Check(level SafetyLevel) error {
	// Read operations are always allowed.
	if level == ReadLevel {
		return nil
	}

	// --dry-run is always allowed (no mutation sent).
	if p.DryRun {
		return nil
	}

	// --read-only blocks all writes, even with --confirm.
	if p.ReadOnly {
		return ErrSafetyBlocked
	}

	// CANVAS_READ_ONLY=1 env var also blocks writes.
	if os.Getenv("CANVAS_READ_ONLY") == "1" {
		return ErrSafetyBlocked
	}

	// Destructive operations require --confirm-delete.
	if level == Destructive {
		if !p.ConfirmDelete {
			return ErrNeedsConfirmDelete
		}
		return nil
	}

	// Low-risk and high-risk writes require --confirm.
	if !p.Confirm {
		return ErrNeedsConfirm
	}

	return nil
}
