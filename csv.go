package main

import (
	"encoding/csv"
	"fmt"
	"io"
)

func ReadCSVFile(reader io.Reader, delimiter rune) TabularData {
	csv := csv.NewReader(reader)

	data := TabularData{
		Width: 0,
		Rows:  make([][]string, 0, 100),
	}

	csv.Comma = delimiter
	if headers, err := csv.Read(); err != nil {
		// TODO: error handling
		panic(err)
	} else {
		columns := make([]Column, len(headers))
		for i, col := range headers {
			columns[i] = Column{Name: col, Width: len(col)}
			data.Width += len(col)
		}
		data.Columns = columns
	}

	for {
		record, err := csv.Read()

		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println("Error:", err)
			panic(err)
		}

		if len(record) != len(data.Columns) {
			fmt.Println("INVALID ROW")
			panic("ASDF")
		}

		data.Rows = append(data.Rows, record)

		for i, col := range record {
			if len(col) > data.Columns[i].Width {
				data.Columns[i].Width = len(col)
			}
		}
	}

	return data
}
