package main

import (
	"strings"
	"regexp"
	"fmt"
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
	filter, value string
	valueFloat    float64
	cmpType       ComparisonType
	colIdx        int
}

var CMP_OP_REGEX = regexp.MustCompile(`^(.+)([!=><]+)(.+)$`)

// parse a filter string into an instance of the Filter interface
func parseFilter(fs string) Filter {
	if match := CMP_OP_REGEX.FindStringSubmatch(fs); len(match) > 0 {
		var cmpType ComparisonType

		fmt.Printf("match is: %v\n", match)

		switch match[2] {
		case "=", "==":
			cmpType = CmpEq
		case "!=":
			cmpType = CmpNeq
		case ">":
			cmpType = CmpGt
		case ">=":
			cmpType = CmpGte
		case "<":
			cmpType = CmpLt
		case "<=":
			cmpType = CmpLte
		default:

		}

		fmt.Printf("cmptype is %v\n", cmpType)

		return ColumnFilter{}
	} else {
		return RowFilter{
			filter: fs,
			caseSensitive: false,
		}
	}
}

func (f ColumnFilter) matches(row []string) bool {
	var v1, v2 float64

	switch f.cmpType {
	case CmpEq:
		return v1 == v2
	case CmpNeq:
		return v1 != v2
	case CmpGt:
		return v1 > v2
	case CmpGte:
		return v1 >= v2
	case CmpLt:
		return v1 < v2
	case CmpLte:
		return v1 <= v2
	}

	lowerFilter := strings.ToLower(f.filter)
	lowerCol := strings.ToLower(row[f.colIdx])

	return strings.Contains(lowerCol, lowerFilter)
}
