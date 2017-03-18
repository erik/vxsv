package vxsv

import (
	"encoding/csv"
	"errors"
	"io"
)

func ReadCSVFile(reader io.Reader, delimiter rune, count int64) (*TabularData, error) {
	csv := csv.NewReader(reader)

	data := &TabularData{
		Rows: make([][]string, 0, 100),
	}

	csv.Comma = delimiter
	if headers, err := csv.Read(); err == nil {
		columns := make([]Column, len(headers))
		for i, col := range headers {
			width := clamp(len(col), 1, len(col))
			columns[i] = Column{Name: col, Width: width}
		}

		data.Columns = columns
	} else {
		return nil, err
	}

	var i int64
	for i = 0; i < count; i++ {
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

		for j, col := range record {
			if len(col) > data.Columns[j].Width {
				data.Columns[j].Width = len(col)
			}
		}
	}

	return data, nil
}
