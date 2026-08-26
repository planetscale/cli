package sql

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/sqlquery"
)

// printHumanResult prints a query result for the human output format: rows as
// a mysql-style table, or one column per line when vertical is set.
func printHumanResult(ch *cmdutil.Helper, result *sqlquery.Result, vertical bool) {
	if result.RowsAffected > 0 && result.RowCount == 0 {
		ch.Printer.Printf("Rows affected: %d\n", result.RowsAffected)
		return
	}
	if result.RowCount > 0 {
		var b strings.Builder
		if vertical {
			renderVertical(&b, result.Columns, result.Rows)
		} else {
			renderTable(&b, result.Columns, result.Rows)
		}
		ch.Printer.Printf("%s", b.String())
	}
	ch.Printer.Printf("Returned %d row(s)\n", result.RowCount)
}

// stripVerticalTerminator removes a trailing \G or \g client terminator from a
// query. These are mysql client constructs, not SQL: servers reject them with a
// syntax error. A trailing \G also requests vertical output, mirroring the
// mysql client, so the second return value reports whether it was present.
func stripVerticalTerminator(query string) (string, bool) {
	trimmed := strings.TrimRight(query, " \t\r\n")
	if strings.HasSuffix(trimmed, `\G`) {
		return strings.TrimRight(trimmed[:len(trimmed)-2], " \t\r\n"), true
	}
	if strings.HasSuffix(trimmed, `\g`) {
		return strings.TrimRight(trimmed[:len(trimmed)-2], " \t\r\n"), false
	}
	return query, false
}

func formatValue(v any) string {
	if v == nil {
		return "NULL"
	}
	return fmt.Sprintf("%v", v)
}

// cellWidth returns the display width of a value, using its longest line so
// multi-line values (e.g. GTID sets) don't blow up the whole column.
func cellWidth(s string) int {
	width := 0
	for _, line := range strings.Split(s, "\n") {
		if w := utf8.RuneCountInString(line); w > width {
			width = w
		}
	}
	return width
}

// renderTable writes rows as a mysql-style ASCII table, with columns in the
// order the server returned them.
func renderTable(w io.Writer, columns []string, rows []map[string]any) {
	widths := make([]int, len(columns))
	for i, col := range columns {
		widths[i] = utf8.RuneCountInString(col)
	}
	cells := make([][]string, len(rows))
	for r, row := range rows {
		cells[r] = make([]string, len(columns))
		for i, col := range columns {
			s := formatValue(row[col])
			cells[r][i] = s
			if cw := cellWidth(s); cw > widths[i] {
				widths[i] = cw
			}
		}
	}

	var border strings.Builder
	for _, width := range widths {
		border.WriteString("+")
		border.WriteString(strings.Repeat("-", width+2))
	}
	border.WriteString("+\n")

	fmt.Fprint(w, border.String())
	writeTableRow(w, columns, widths)
	fmt.Fprint(w, border.String())
	for _, row := range cells {
		writeTableRow(w, row, widths)
	}
	fmt.Fprint(w, border.String())
}

// writeTableRow writes one logical row. A cell with embedded newlines spreads
// the row over multiple physical lines, padding every column on every line so
// the grid stays closed.
func writeTableRow(w io.Writer, cells []string, widths []int) {
	lines := make([][]string, len(cells))
	height := 1
	for i, cell := range cells {
		lines[i] = strings.Split(cell, "\n")
		if len(lines[i]) > height {
			height = len(lines[i])
		}
	}
	for line := 0; line < height; line++ {
		for i := range cells {
			var s string
			if line < len(lines[i]) {
				s = lines[i][line]
			}
			pad := widths[i] - utf8.RuneCountInString(s)
			if pad < 0 {
				pad = 0
			}
			fmt.Fprintf(w, "| %s%s ", s, strings.Repeat(" ", pad))
		}
		fmt.Fprint(w, "|\n")
	}
}

// renderVertical writes rows in the mysql \G style: one column per line,
// column names right-aligned, in the order the server returned them.
func renderVertical(w io.Writer, columns []string, rows []map[string]any) {
	nameWidth := 0
	for _, col := range columns {
		if l := utf8.RuneCountInString(col); l > nameWidth {
			nameWidth = l
		}
	}
	for i, row := range rows {
		fmt.Fprintf(w, "*************************** %d. row ***************************\n", i+1)
		for _, col := range columns {
			fmt.Fprintf(w, "%*s: %s\n", nameWidth, col, formatValue(row[col]))
		}
	}
}
