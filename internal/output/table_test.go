package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestTable_RenderBasicTable(t *testing.T) {
	table := Table{
		Headers: []string{"ID", "Name"},
		Rows: [][]string{
			{"1", "Alice"},
			{"2", "Bob"},
		},
	}

	var buf bytes.Buffer
	if err := table.Render(&buf, true); err != nil {
		t.Fatalf("Render error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "ID") {
		t.Error("output missing header ID")
	}
	if !strings.Contains(output, "Name") {
		t.Error("output missing header Name")
	}
	if !strings.Contains(output, "Alice") {
		t.Error("output missing row Alice")
	}
	if !strings.Contains(output, "Bob") {
		t.Error("output missing row Bob")
	}
}

func TestTable_ColumnAlignment(t *testing.T) {
	table := Table{
		Headers: []string{"ID", "Name"},
		Rows: [][]string{
			{"1", "Alice"},
			{"123", "Bob"},
		},
	}

	var buf bytes.Buffer
	if err := table.Render(&buf, true); err != nil {
		t.Fatalf("Render error: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines (header + separator + rows), got %d", len(lines))
	}

	// All lines should have the same length for aligned columns
	expectedLen := len(lines[0])
	for i, line := range lines {
		if len(line) != expectedLen {
			t.Errorf("line %d has length %d, want %d: %q", i, len(line), expectedLen, line)
		}
	}
}

func TestTable_EmptyData(t *testing.T) {
	table := Table{
		Headers: []string{"ID", "Name"},
		Rows:    [][]string{},
	}

	var buf bytes.Buffer
	if err := table.Render(&buf, true); err != nil {
		t.Fatalf("Render error: %v", err)
	}

	output := buf.String()
	// Should still render headers even with no rows
	if !strings.Contains(output, "ID") {
		t.Error("output missing headers for empty table")
	}
	if !strings.Contains(output, "Name") {
		t.Error("output missing headers for empty table")
	}
}

func TestTable_NoColorMode(t *testing.T) {
	table := Table{
		Headers: []string{"ID", "Name"},
		Rows: [][]string{
			{"1", "Test"},
		},
	}

	var buf bytes.Buffer
	if err := table.Render(&buf, true); err != nil {
		t.Fatalf("Render error: %v", err)
	}

	output := buf.String()
	// No ANSI escape codes in no-color mode
	if strings.Contains(output, "\033[") {
		t.Error("no-color output contains ANSI escape codes")
	}
}

func TestTable_SeparatorLine(t *testing.T) {
	table := Table{
		Headers: []string{"ID", "Name"},
		Rows: [][]string{
			{"1", "Test"},
		},
	}

	var buf bytes.Buffer
	if err := table.Render(&buf, true); err != nil {
		t.Fatalf("Render error: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}

	// Second line should be a separator made of dashes
	sepLine := lines[1]
	if !strings.Contains(sepLine, "---") {
		t.Errorf("expected separator line with dashes, got: %q", sepLine)
	}
}

func TestTable_UnicodeContent(t *testing.T) {
	table := Table{
		Headers: []string{"Name"},
		Rows: [][]string{
			{"Jose"},
			{"Muller"},
		},
	}

	var buf bytes.Buffer
	if err := table.Render(&buf, true); err != nil {
		t.Fatalf("Render error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Jose") {
		t.Error("output missing Jose")
	}
	if !strings.Contains(output, "Muller") {
		t.Error("output missing Muller")
	}
}
