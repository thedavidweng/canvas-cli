package output

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Table holds column headers and row data for human-readable output.
type Table struct {
	Headers []string
	Rows    [][]string
}

// Render writes a formatted ASCII table to w. Headers are bold when the output
// is a terminal and noColor is false; otherwise no ANSI escape codes are emitted.
func (t Table) Render(w io.Writer, noColor bool) error {
	if len(t.Headers) == 0 {
		return nil
	}

	color := !noColor && isTerminalWriter(w)
	numCols := len(t.Headers)
	colWidths := make([]int, numCols)

	for i, h := range t.Headers {
		colWidths[i] = len(h)
	}

	for _, row := range t.Rows {
		for i, cell := range row {
			if i < numCols && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	for i, h := range t.Headers {
		if i > 0 {
			fmt.Fprint(w, "  ")
		}
		if color {
			fmt.Fprintf(w, "\033[1m%-*s\033[0m", colWidths[i], h)
		} else {
			fmt.Fprintf(w, "%-*s", colWidths[i], h)
		}
	}
	fmt.Fprintln(w)

	for i := range t.Headers {
		if i > 0 {
			fmt.Fprint(w, "  ")
		}
		fmt.Fprint(w, strings.Repeat("-", colWidths[i]))
	}
	fmt.Fprintln(w)

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

func isTerminalWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
