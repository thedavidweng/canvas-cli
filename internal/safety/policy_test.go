package safety

import (
	"errors"
	"testing"
)

func TestSafetyLevelConstants(t *testing.T) {
	if ReadLevel != 0 {
		t.Errorf("ReadLevel = %d, want 0", ReadLevel)
	}
	if LowRiskWrite != 1 {
		t.Errorf("LowRiskWrite = %d, want 1", LowRiskWrite)
	}
	if HighRiskWrite != 2 {
		t.Errorf("HighRiskWrite = %d, want 2", HighRiskWrite)
	}
}

func TestCheck_ReadLevel_AlwaysAllowed(t *testing.T) {
	cases := []struct {
		name   string
		policy Policy
	}{
		{"default policy", NewPolicy(false, false, false)},
		{"read-only", NewPolicy(true, false, false)},
		{"dry-run", NewPolicy(false, true, false)},
		{"confirm", NewPolicy(false, false, true)},
		{"all flags", NewPolicy(true, true, true)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.policy.Check(ReadLevel); err != nil {
				t.Errorf("ReadLevel should always be allowed, got error: %v", err)
			}
		})
	}
}

func TestCheck_LowRiskWrite_NeedsConfirm(t *testing.T) {
	p := NewPolicy(false, false, false)
	err := p.Check(LowRiskWrite)
	if err == nil {
		t.Fatal("expected error when LowRiskWrite without --confirm")
	}
	if !errors.Is(err, ErrNeedsConfirm) {
		t.Errorf("expected ErrNeedsConfirm, got %v", err)
	}
}

func TestCheck_LowRiskWrite_WithConfirm(t *testing.T) {
	p := NewPolicy(false, false, true)
	if err := p.Check(LowRiskWrite); err != nil {
		t.Errorf("LowRiskWrite with --confirm should succeed, got: %v", err)
	}
}

func TestCheck_HighRiskWrite_NeedsConfirm(t *testing.T) {
	p := NewPolicy(false, false, false)
	err := p.Check(HighRiskWrite)
	if err == nil {
		t.Fatal("expected error when HighRiskWrite without --confirm")
	}
	if !errors.Is(err, ErrNeedsConfirm) {
		t.Errorf("expected ErrNeedsConfirm, got %v", err)
	}
}

func TestCheck_HighRiskWrite_WithConfirm(t *testing.T) {
	p := NewPolicy(false, false, true)
	if err := p.Check(HighRiskWrite); err != nil {
		t.Errorf("HighRiskWrite with --confirm should succeed, got: %v", err)
	}
}

func TestCheck_HighRiskWrite_DryRunAllowed(t *testing.T) {
	p := NewPolicy(false, true, false)
	if err := p.Check(HighRiskWrite); err != nil {
		t.Errorf("HighRiskWrite with --dry-run should succeed, got: %v", err)
	}
}

func TestCheck_ReadOnly_BlocksAllWrites(t *testing.T) {
	levels := []SafetyLevel{LowRiskWrite, HighRiskWrite}
	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			p := NewPolicy(true, false, false)
			err := p.Check(level)
			if err == nil {
				t.Fatal("expected error for write under --read-only")
			}
			if !errors.Is(err, ErrSafetyBlocked) {
				t.Errorf("expected ErrSafetyBlocked, got %v", err)
			}
			var se *SafetyError
			if !errors.As(err, &se) {
				t.Fatalf("expected *SafetyError, got %T", err)
			}
			if se.ExitCode != 7 {
				t.Errorf("exit code = %d, want 7", se.ExitCode)
			}
		})
	}
}

// CANVAS_READ_ONLY is resolved by the config layer into ResolvedConfig.ReadOnly
// and passed to NewPolicy. Policy.Check must not read os.Getenv directly.
func TestCheck_EnvVarDoesNotInfluencePolicy(t *testing.T) {
	t.Setenv("CANVAS_READ_ONLY", "1")

	p := NewPolicy(false, false, true)
	err := p.Check(LowRiskWrite)
	if err != nil {
		t.Fatalf("Policy{ReadOnly:false} with --confirm should allow LowRiskWrite even when CANVAS_READ_ONLY=1 is set, got: %v", err)
	}
}

func TestCheck_ReadOnlyOverridesConfirm(t *testing.T) {
	p := NewPolicy(true, false, true)
	levels := []SafetyLevel{LowRiskWrite, HighRiskWrite}
	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			err := p.Check(level)
			if err == nil {
				t.Fatal("expected --read-only to override --confirm")
			}
			if !errors.Is(err, ErrSafetyBlocked) {
				t.Errorf("expected ErrSafetyBlocked, got %v", err)
			}
		})
	}
}

func TestCheck_DryRunAllowedUnderReadOnly(t *testing.T) {
	levels := []SafetyLevel{LowRiskWrite, HighRiskWrite}
	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			p := NewPolicy(true, true, false)
			if err := p.Check(level); err != nil {
				t.Errorf("--dry-run should be allowed under --read-only for %s, got: %v", level, err)
			}
		})
	}
}

func TestCheck_DryRun_AllowedWithoutConfirm(t *testing.T) {
	levels := []SafetyLevel{LowRiskWrite, HighRiskWrite}
	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			p := NewPolicy(false, true, false)
			if err := p.Check(level); err != nil {
				t.Errorf("--dry-run should be allowed without --confirm for %s, got: %v", level, err)
			}
		})
	}
}

func TestSafetyError_ErrorString(t *testing.T) {
	err := &SafetyError{Message: "operation blocked by read-only mode", ExitCode: 7}
	expected := "operation blocked by read-only mode"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestErrSafetyBlocked_IsSafetyError(t *testing.T) {
	var se *SafetyError
	if !errors.As(ErrSafetyBlocked, &se) {
		t.Fatal("ErrSafetyBlocked should be a *SafetyError")
	}
	if se.ExitCode != 7 {
		t.Errorf("ErrSafetyBlocked exit code = %d, want 7", se.ExitCode)
	}
}

func TestErrNeedsConfirm_IsSafetyError(t *testing.T) {
	var se *SafetyError
	if !errors.As(ErrNeedsConfirm, &se) {
		t.Fatal("ErrNeedsConfirm should be a *SafetyError")
	}
	if se.ExitCode != 7 {
		t.Errorf("ErrNeedsConfirm exit code = %d, want 7 (safety blocked)", se.ExitCode)
	}
}

func TestSafetyLevel_String_Unknown(t *testing.T) {
	unknown := SafetyLevel(99)
	if got := unknown.String(); got != "Unknown" {
		t.Errorf("SafetyLevel(99).String() = %q, want %q", got, "Unknown")
	}
}

func TestNewPolicy_Fields(t *testing.T) {
	p := NewPolicy(true, true, true)
	if !p.ReadOnly {
		t.Error("ReadOnly should be true")
	}
	if !p.DryRun {
		t.Error("DryRun should be true")
	}
	if !p.Confirm {
		t.Error("Confirm should be true")
	}

	p2 := NewPolicy(false, false, false)
	if p2.ReadOnly || p2.DryRun || p2.Confirm {
		t.Error("all fields should be false")
	}
}
