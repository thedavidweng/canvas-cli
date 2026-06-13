// Package audit provides append-only JSONL audit logging for canvas-cli mutations.
package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
)

// Auditor writes AuditEvent records to a local JSONL file.
type Auditor struct {
	path    string
	enabled bool
	mu      sync.Mutex
}

// NewAuditor creates an Auditor that writes to path when enabled is true.
// When enabled is false, all WriteEvent calls are no-ops.
func NewAuditor(path string, enabled bool) *Auditor {
	return &Auditor{
		path:    path,
		enabled: enabled,
	}
}

// WriteEvent appends a single JSONL line to the audit log file.
// If the auditor is disabled, this is a no-op and returns nil.
func (a *Auditor) WriteEvent(event canvas.AuditEvent) error {
	if !a.enabled {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Ensure parent directory exists.
	dir := filepath.Dir(a.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("audit: create dir: %w", err)
	}

	f, err := os.OpenFile(a.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("audit: open file: %w", err)
	}
	defer f.Close()

	line, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("audit: marshal event: %w", err)
	}
	line = append(line, '\n')

	if _, err := f.Write(line); err != nil {
		return fmt.Errorf("audit: write event: %w", err)
	}

	return nil
}

// DefaultPath returns the default audit log file path:
// ${XDG_STATE_HOME:-~/.local/state}/canvas-cli/audit.jsonl
func DefaultPath() string {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "canvas-cli", "audit.jsonl")
}

// HashBody returns the SHA-256 hex digest of body, prefixed with "sha256:".
func HashBody(body string) string {
	h := sha256.Sum256([]byte(body))
	return "sha256:" + hex.EncodeToString(h[:])
}
