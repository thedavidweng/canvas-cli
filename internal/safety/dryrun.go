package safety

import (
	"fmt"
	"strings"
)

// Preview holds the information needed to display a dry-run summary.
type Preview struct {
	Method         string
	Path           string
	ResourceIDs    []string
	PayloadSummary string
}

// FormatPreview renders a dry-run preview showing method, path, resource IDs,
// and a payload summary. The payload summary is truncated to 200 characters.
func FormatPreview(p Preview) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("DRY RUN: %s %s\n", p.Method, p.Path))

	if len(p.ResourceIDs) > 0 {
		b.WriteString(fmt.Sprintf("Resource IDs: %s\n", strings.Join(p.ResourceIDs, ", ")))
	}

	if p.PayloadSummary != "" {
		summary := p.PayloadSummary
		const maxLen = 200
		if len(summary) > maxLen {
			summary = summary[:maxLen] + "... (truncated)"
		}
		b.WriteString(fmt.Sprintf("Payload: %s\n", summary))
	}

	b.WriteString("No mutation sent.")

	return b.String()
}
