package main

import (
	"encoding/csv"
	"errors"
	"io"
)

func ReadCSVFile(reader io.Reader, delimiter rune) (*TabularData, error) {
	csv := csv.NewReader(reader)

	data := &TabularData{
		Width: 0,
		Rows:  make([][]string, 0, 100),
	}

	csv.Comma = delimiter
	if headers, err := csv.Read(); err != nil {
		return nil, err
	} else {
		columns := make([]Column, len(headers))
		for i, col := range headers {
			width := clamp(len(col), 1, len(col))
			columns[i] = Column{Name: col, Width: width}
			data.Width += width
		}
		data.Columns = columns
	}

	for {
		record, err := csv.Read()

		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		if len(record) != len(data.Columns) {
			return nil, errors.New("Row has incorrect number of columns")
		}

		data.Rows = append(data.Rows, record)

		for i, col := range record {
			if len(col) > data.Columns[i].Width {
				data.Columns[i].Width = len(col)
			}
		}
	}

	return data, nil
}
