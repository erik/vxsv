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
	columns, spans := parseColumns(columnString)

	// Skip the horizontal line
	scanner.Scan()
	width := len(scanner.Text())

	rows := [][]string{}

	for scanner.Scan() {
		// This is the last line that's printed, e.g. (100 rows)
		if scanner.Text()[0] == '(' {
			break
		}

		rows = append(rows, parseRow(spans, scanner.Text()))
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

func parseColumns(columnString string) ([]string, [][]int) {
	split := strings.Split(columnString, " | ")

	columns := make([]string, len(split))
	spans := make([][]int, len(split))

	last_offset := 0

	for i, col := range split {
		columns[i] = strings.TrimSpace(col)
		spans[i] = []int{last_offset, last_offset + len(col)}

		last_offset += len(col) + 3
	}

	// Make sure we skip the leading space
	if last_offset > 0 {
		spans[0][0] += 1
	}

	return columns, spans
}

func parseRow(spans [][]int, str string) []string {
	row := make([]string, len(spans))

	for i, span := range spans {
		// Make sure we don't over shoot the string length
		if span[1] >= len(str) {
			row[i] = str[span[0]:len(str)]
		} else {
			row[i] = str[span[0]:span[1]]
		}
	}

	return row
}
