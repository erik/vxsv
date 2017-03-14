package vxsv

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type Filter interface {
	String() string
	Matches(row []string) bool
}

type EmptyFilter struct{}

func (f EmptyFilter) String() string        { return "" }
func (f EmptyFilter) Matches([]string) bool { return true }

type RowFilter struct {
	filter        string
	caseSensitive bool
}

func (f RowFilter) String() string { return f.filter }
func (f RowFilter) Matches(row []string) bool {
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
	expression string
	value      string
	valueFloat float64
	cmpType    ComparisonType
	colIdx     int
}

const OP_CHARS = "!=><"

var CMP_OP_REGEX = regexp.MustCompile(`^(.+?)([!=><]+)(.+)$`)

// parse a filter string into an instance of the Filter interface
func (ui *UI) parseFilter(fs string) (Filter, error) {
	if !strings.ContainsAny(fs, OP_CHARS) {
		return RowFilter{
			filter:        fs,
			caseSensitive: false,
		}, nil
	}

	if match := CMP_OP_REGEX.FindStringSubmatch(fs); len(match) > 0 {
		filter := ColumnFilter{
			expression: fs,
		}

		var (
			column = strings.TrimSpace(match[1])
			oper   = match[2]
			value  = strings.TrimSpace(match[3])
		)

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
	}

	return nil, fmt.Errorf("Filter didn't match expected format: %v", CMP_OP_REGEX)
}

func (f ColumnFilter) String() string { return f.expression }
func (f ColumnFilter) Matches(row []string) bool {
	valStr := row[f.colIdx]

	if math.IsNaN(f.valueFloat) {
		switch f.cmpType {
		case CmpEq:
			return valStr == f.value
		case CmpNeq:
			return valStr != f.value
		case CmpGt:
			return valStr > f.value
		case CmpGte:
			return valStr >= f.value
		case CmpLt:
			return valStr < f.value
		case CmpLte:
			return valStr <= f.value
		}
	} else if val, err := strconv.ParseFloat(strings.TrimSpace(valStr), 64); err == nil {
		switch f.cmpType {
		case CmpEq:
			return val == f.valueFloat
		case CmpNeq:
			return val != f.valueFloat
		case CmpGt:
			return val > f.valueFloat
		case CmpGte:
			return val >= f.valueFloat
		case CmpLt:
			return val < f.valueFloat
		case CmpLte:
			return val <= f.valueFloat
		}
	}

	return false
}
