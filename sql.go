package main

import (
	"bufio"
	"io"
	"strings"
)

func ReadPsqlTable(reader io.Reader) (*TabularData, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Scan()

	columnString := scanner.Text()
	columns := parseColumns(columnString)

	// Skip the horizontal line
	scanner.Scan()

	rows := [][]string{}

	for scanner.Scan() {
		// This is the last line that's printed, e.g. (100 rows)
		if scanner.Text()[0] == '(' {
			break
		}

		rows = append(rows, parseRow(columns, scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &TabularData{
		Columns: columns,
		Rows:    rows,
	}, nil
}

// Parses MySQL output format:
//
// +------+------+------+
// | colA | colB | colC |
// +------+------+------+
// | foo  | bar  | baz  |
// | foo2 | bar2 | baz2 |
// +------+------+------+
// 2 rows in set
func ReadMySqlTable(reader io.Reader) (*TabularData, error) {
	scanner := bufio.NewScanner(reader)

	// Skip leading horizontal line
	scanner.Scan()

	scanner.Scan()
	columnString := scanner.Text()
	columns := parseColumns(columnString[1 : len(columnString)-2])

	// Skip trailing horizontal line
	scanner.Scan()

	rows := [][]string{}
	for scanner.Scan() {
		row := scanner.Text()

		// last line
		if row[0] == '+' {
			break
		}

		rows = append(rows, parseRow(columns, row[1:len(row)-2]))
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &TabularData{
		Columns: columns,
		Rows:    rows,
	}, nil
}

func parseColumns(columnString string) []Column {
	split := strings.Split(columnString, " | ")

	columns := make([]Column, len(split))

	for i, col := range split {
		columns[i] = Column{
			Name:  strings.TrimSpace(col),
			Width: len(col),
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

		row[i] = strings.TrimSpace(row[i])

		offset += col.Width + 3
	}

	return row
}
