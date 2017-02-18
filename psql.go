package main

import (
	"bufio"
	"io"
	"strings"
)

func ReadPsqlTable(reader io.Reader) TabularData {
	scanner := bufio.NewScanner(reader)
	scanner.Scan()

	columnString := scanner.Text()
	columns := parseColumns(columnString)

	// Skip the horizontal line
	scanner.Scan()
	width := len(scanner.Text())

	rows := [][]string{}

	for scanner.Scan() {
		// This is the last line that's printed, e.g. (100 rows)
		if scanner.Text()[0] == '(' {
			break
		}

		rows = append(rows, parseRow(columns, scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		// TODO: scream and shout...
	}

	return TabularData{
		Width:   width,
		Columns: columns,
		Rows:    rows,
	}
}

func parseColumns(columnString string) []Column {
	split := strings.Split(columnString, " | ")

	columns := make([]Column, len(split))

	for i, col := range split {
		columns[i] = Column{
			Name:      strings.TrimSpace(col),
			Width:     len(col),
			Collapsed: false,
		}
	}

	// Make sure we skip the leading space in the first column
	columns[0].Width -= 1

	return columns
}

// TODO: doesn't handle multi-line rows
func parseRow(columns []Column, str string) []string {
	row := make([]string, len(columns))

	// Skip leading space
	offset := 1

	for i, col := range columns {
		// Make sure we don't over shoot the string length
		if offset+col.Width >= len(str) {
			row[i] = str[offset:len(str)]
		} else {
			row[i] = str[offset : offset+col.Width]
		}

		offset += col.Width + 3
	}

	return row
}
