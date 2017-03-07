package main

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type Filter interface {
	matches(row []string) bool
}

type RowFilter struct {
	filter        string
	caseSensitive bool
}

func (f RowFilter) matches(row []string) bool {
	if f.filter == "" {
		return true
	}

	for _, col := range row {
		if f.caseSensitive && strings.Contains(col, f.filter) {
			return true
		} else if !f.caseSensitive {
			lowerFilter := strings.ToLower(f.filter)
			lowerCol := strings.ToLower(col)
			if strings.Contains(lowerCol, lowerFilter) {
				return true
			}
		}
	}
	return false
}

type ComparisonType int

const (
	CmpEq = iota
	CmpNeq
	CmpGt
	CmpGte
	CmpLt
	CmpLte
)

type ColumnFilter struct {
	value      string
	valueFloat float64
	cmpType    ComparisonType
	colIdx     int
}

var CMP_OP_REGEX = regexp.MustCompile(`^(.+)([!=><]+)(.+)$`)

// parse a filter string into an instance of the Filter interface
func (ui *UI) parseFilter(fs string) (Filter, error) {
	if match := CMP_OP_REGEX.FindStringSubmatch(fs); len(match) > 0 {
		filter := ColumnFilter{}

		var (
			column = strings.TrimSpace(match[1])
			oper   = match[2]
			value  = strings.TrimSpace(match[3])
		)

		fmt.Printf("match is: %v\n", match)

		filter.colIdx = -1
		for i, col := range ui.columns {
			if col == column {
				filter.colIdx = i
				break
			}
		}

		if filter.colIdx == -1 {
			return nil, fmt.Errorf("No such column: \"%s\"", column)
		}

		switch oper {
		case "=", "==":
			filter.cmpType = CmpEq
		case "!=":
			filter.cmpType = CmpNeq
		case ">":
			filter.cmpType = CmpGt
		case ">=":
			filter.cmpType = CmpGte
		case "<":
			filter.cmpType = CmpLt
		case "<=":
			filter.cmpType = CmpLte
		default:
			return nil, fmt.Errorf("No such comparison operation: \"%s\"", oper)
		}

		filter.value = value
		if val, err := strconv.ParseFloat(value, 64); err == nil {
			filter.valueFloat = val
		} else {
			filter.valueFloat = math.NaN()
		}

		return filter, nil
	} else {
		return RowFilter{
			filter:        fs,
			caseSensitive: false,
		}, nil
	}
}

func (f ColumnFilter) matches(row []string) bool {
	valStr := row[f.colIdx]

	if math.IsNaN(f.valueFloat) {
		switch f.cmpType {
		case CmpEq:
			return f.value == valStr
		case CmpNeq:
			return f.value != valStr
		case CmpGt:
			return f.value > valStr
		case CmpGte:
			return f.value >= valStr
		case CmpLt:
			return f.value < valStr
		case CmpLte:
			return f.value <= valStr
		}
	} else if val, err := strconv.ParseFloat(strings.TrimSpace(valStr), 64); err == nil {
		switch f.cmpType {
		case CmpEq:
			return f.valueFloat == val
		case CmpNeq:
			return f.valueFloat != val
		case CmpGt:
			return f.valueFloat > val
		case CmpGte:
			return f.valueFloat >= val
		case CmpLt:
			return f.valueFloat < val
		case CmpLte:
			return f.valueFloat <= val
		}
	}

	return false
}
