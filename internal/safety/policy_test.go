package safety

import (
	"errors"
	"testing"
)

// --- SafetyLevel constants ---

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
	if Destructive != 3 {
		t.Errorf("Destructive = %d, want 3", Destructive)
	}
}

// --- ReadLevel always allowed ---

func TestCheck_ReadLevel_AlwaysAllowed(t *testing.T) {
	cases := []struct {
		name   string
		policy Policy
	}{
		{"default policy", NewPolicy(false, false, false, false)},
		{"read-only", NewPolicy(true, false, false, false)},
		{"dry-run", NewPolicy(false, true, false, false)},
		{"confirm", NewPolicy(false, false, true, false)},
		{"confirm-delete", NewPolicy(false, false, false, true)},
		{"all flags", NewPolicy(true, true, true, true)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.policy.Check(ReadLevel); err != nil {
				t.Errorf("ReadLevel should always be allowed, got error: %v", err)
			}
		})
	}
}

// --- LowRiskWrite needs --confirm ---

func TestCheck_LowRiskWrite_NeedsConfirm(t *testing.T) {
	p := NewPolicy(false, false, false, false)
	err := p.Check(LowRiskWrite)
	if err == nil {
		t.Fatal("expected error when LowRiskWrite without --confirm")
	}
	if !errors.Is(err, ErrNeedsConfirm) {
		t.Errorf("expected ErrNeedsConfirm, got %v", err)
	}
}

func TestCheck_LowRiskWrite_WithConfirm(t *testing.T) {
	p := NewPolicy(false, false, true, false)
	if err := p.Check(LowRiskWrite); err != nil {
		t.Errorf("LowRiskWrite with --confirm should succeed, got: %v", err)
	}
}

// --- HighRiskWrite needs --confirm, defaults to dry-run ---

func TestCheck_HighRiskWrite_NeedsConfirm(t *testing.T) {
	p := NewPolicy(false, false, false, false)
	err := p.Check(HighRiskWrite)
	if err == nil {
		t.Fatal("expected error when HighRiskWrite without --confirm")
	}
	if !errors.Is(err, ErrNeedsConfirm) {
		t.Errorf("expected ErrNeedsConfirm, got %v", err)
	}
}

func TestCheck_HighRiskWrite_WithConfirm(t *testing.T) {
	p := NewPolicy(false, false, true, false)
	if err := p.Check(HighRiskWrite); err != nil {
		t.Errorf("HighRiskWrite with --confirm should succeed, got: %v", err)
	}
}

func TestCheck_HighRiskWrite_DryRunAllowed(t *testing.T) {
	p := NewPolicy(false, true, false, false)
	if err := p.Check(HighRiskWrite); err != nil {
		t.Errorf("HighRiskWrite with --dry-run should succeed, got: %v", err)
	}
}

// --- Destructive needs --confirm-delete ---

func TestCheck_Destructive_NeedsConfirmDelete(t *testing.T) {
	p := NewPolicy(false, false, false, false)
	err := p.Check(Destructive)
	if err == nil {
		t.Fatal("expected error when Destructive without --confirm-delete")
	}
	if !errors.Is(err, ErrNeedsConfirmDelete) {
		t.Errorf("expected ErrNeedsConfirmDelete, got %v", err)
	}
}

func TestCheck_Destructive_WithConfirmOnly(t *testing.T) {
	p := NewPolicy(false, false, true, false)
	err := p.Check(Destructive)
	if err == nil {
		t.Fatal("expected error when Destructive with --confirm but no --confirm-delete")
	}
	if !errors.Is(err, ErrNeedsConfirmDelete) {
		t.Errorf("expected ErrNeedsConfirmDelete, got %v", err)
	}
}

func TestCheck_Destructive_WithConfirmDelete(t *testing.T) {
	p := NewPolicy(false, false, false, true)
	if err := p.Check(Destructive); err != nil {
		t.Errorf("Destructive with --confirm-delete should succeed, got: %v", err)
	}
}

func TestCheck_Destructive_DryRunAllowed(t *testing.T) {
	p := NewPolicy(false, true, false, false)
	if err := p.Check(Destructive); err != nil {
		t.Errorf("Destructive with --dry-run should succeed, got: %v", err)
	}
}

// --- --read-only blocks ALL writes (exit code 7) ---

func TestCheck_ReadOnly_BlocksAllWrites(t *testing.T) {
	levels := []SafetyLevel{LowRiskWrite, HighRiskWrite, Destructive}
	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			p := NewPolicy(true, false, false, false)
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

// --- CANVAS_READ_ONLY env var should NOT be read by Policy.Check ---
// The config layer resolves CANVAS_READ_ONLY into ResolvedConfig.ReadOnly,
// which is passed to NewPolicy. Policy.Check must not bypass that by
// reading os.Getenv directly.

func TestCheck_EnvVarDoesNotInfluencePolicy(t *testing.T) {
	t.Setenv("CANVAS_READ_ONLY", "1")

	// Policy says ReadOnly=false (config decided it's not read-only).
	// The env var must have NO effect on Check().
	p := NewPolicy(false, false, true, false)
	err := p.Check(LowRiskWrite)
	if err != nil {
		t.Fatalf("Policy{ReadOnly:false} with --confirm should allow LowRiskWrite even when CANVAS_READ_ONLY=1 is set, got: %v", err)
	}
}

// --- --read-only overrides --confirm (still blocked) ---

func TestCheck_ReadOnlyOverridesConfirm(t *testing.T) {
	p := NewPolicy(true, false, true, true) // read-only + confirm + confirm-delete
	levels := []SafetyLevel{LowRiskWrite, HighRiskWrite, Destructive}
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

// --- --dry-run allowed under --read-only ---

func TestCheck_DryRunAllowedUnderReadOnly(t *testing.T) {
	levels := []SafetyLevel{LowRiskWrite, HighRiskWrite, Destructive}
	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			p := NewPolicy(true, true, false, false) // read-only + dry-run
			if err := p.Check(level); err != nil {
				t.Errorf("--dry-run should be allowed under --read-only for %s, got: %v", level, err)
			}
		})
	}
}

// --- --dry-run sends no mutation (tested via policy allowing it without confirm) ---

func TestCheck_DryRun_AllowedWithoutConfirm(t *testing.T) {
	levels := []SafetyLevel{LowRiskWrite, HighRiskWrite, Destructive}
	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			p := NewPolicy(false, true, false, false)
			if err := p.Check(level); err != nil {
				t.Errorf("--dry-run should be allowed without --confirm for %s, got: %v", level, err)
			}
		})
	}
}

// --- Error type checks ---

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
	if se.ExitCode != 0 {
		t.Errorf("ErrNeedsConfirm exit code = %d, want 0 (caller decides)", se.ExitCode)
	}
}

func TestErrNeedsConfirmDelete_IsSafetyError(t *testing.T) {
	var se *SafetyError
	if !errors.As(ErrNeedsConfirmDelete, &se) {
		t.Fatal("ErrNeedsConfirmDelete should be a *SafetyError")
	}
}

// --- NewPolicy ---

func TestNewPolicy_Fields(t *testing.T) {
	p := NewPolicy(true, true, true, true)
	if !p.ReadOnly {
		t.Error("ReadOnly should be true")
	}
	if !p.DryRun {
		t.Error("DryRun should be true")
	}
	if !p.Confirm {
		t.Error("Confirm should be true")
	}
	if !p.ConfirmDelete {
		t.Error("ConfirmDelete should be true")
	}

	p2 := NewPolicy(false, false, false, false)
	if p2.ReadOnly || p2.DryRun || p2.Confirm || p2.ConfirmDelete {
		t.Error("all fields should be false")
	}
}
