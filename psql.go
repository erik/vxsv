package main

import (
	"bufio"
	"io"
	"strings"
)

func ReadPsqlTable(reader io.Reader) TabularData {
	scanner := bufio.NewScanner(reader)

	columns, offsets := parseColumns(scanner)

	rows := [][]string{}

	for scanner.Scan() {
		rows = append(rows, parseRow(offsets, scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		// TODO: scream and shout...
	}

	return TabularData{
		Columns: columns,
		Rows:    rows,
	}
}

func parseColumns(scanner *bufio.Scanner) ([]string, []int) {
	scanner.Scan()
	split := strings.Split(scanner.Text(), " | ")

	columns := make([]string, len(split))
	offsets := make([]int, len(split))

	last_offset = 0

	for i, col := range split {
		columns[i] = strings.TrimSpace(col)
		offsets[i] = last_offset + len(col)

		last_offset += offsets[i] + 3
	}

	// Eat the horizontal line
	scanner.Scan()

	return columns, offsets
}

func parseRow(offsets []int, str string) []string {
	row := make([]string, len(offsets))
	last_offset := 0

	for i, offset := range offsets {
		row[i] = str[last_offset:offset]

		last_offset += offset
	}

	return row
}
