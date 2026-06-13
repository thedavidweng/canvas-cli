package output

import (
	"fmt"
	"io"
	"strings"
)

// Table holds column headers and row data for human-readable output.
type Table struct {
	Headers []string
	Rows    [][]string
}

// Render writes a formatted ASCII table to w.
// When noColor is true, no ANSI escape codes are emitted.
func (t Table) Render(w io.Writer, noColor bool) error {
	if len(t.Headers) == 0 {
		return nil
	}

	numCols := len(t.Headers)
	colWidths := make([]int, numCols)

	// Compute column widths from headers
	for i, h := range t.Headers {
		colWidths[i] = len(h)
	}

	// Compute column widths from rows
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < numCols && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Write header
	for i, h := range t.Headers {
		if i > 0 {
			fmt.Fprint(w, "  ")
		}
		fmt.Fprintf(w, "%-*s", colWidths[i], h)
	}
	fmt.Fprintln(w)

	// Write separator
	for i := range t.Headers {
		if i > 0 {
			fmt.Fprint(w, "  ")
		}
		fmt.Fprint(w, strings.Repeat("-", colWidths[i]))
	}
	fmt.Fprintln(w)

	// Write rows
	for _, row := range t.Rows {
		for i := 0; i < numCols; i++ {
			if i > 0 {
				fmt.Fprint(w, "  ")
			}
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			fmt.Fprintf(w, "%-*s", colWidths[i], cell)
		}
		fmt.Fprintln(w)
	}

	return nil
}
