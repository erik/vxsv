// Utility functions for rendering text to the terminal.

package vxsv

import (
	"fmt"
	"strconv"

	"github.com/nsf/termbox-go"
)

func writeStringBounded(x, y, bound int, fg, bg termbox.Attribute, msg string) int {
	for _, c := range msg {
		if x >= bound {
			termbox.SetCell(x, y, c, fg, bg)
		}
		x++
	}
	return x
}

func writeString(x, y int, fg, bg termbox.Attribute, msg string) int {
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
		x++
	}
	return x
}

// Fill entire line of screen
func writeLine(x, y int, fg, bg termbox.Attribute, line string) {
	width, _ := termbox.Size()

	for _, c := range line {
		termbox.SetCell(x, y, c, fg, bg)
		x++
	}
	for i := x; i < width; i++ {
		termbox.SetCell(x+i, y, ' ', fg, bg)
	}
}

func (ui *UI) writeModeLine(mode string, left []string) {
	width, height := termbox.Size()

	// Clear the line
	for i := 0; i < width; i++ {
		termbox.SetCell(i, height-1, ' ', termbox.ColorDefault, termbox.ColorDefault)
	}

	var x int
	for _, ch := range mode {
		termbox.SetCell(x, height-1, ch, termbox.ColorDefault|termbox.AttrBold, termbox.ColorDefault)
		x++
	}

	termbox.SetCell(x, height-1, ' ', termbox.ColorDefault, termbox.ColorDefault)
	x++

	for _, str := range left {
		for _, ch := range str {
			termbox.SetCell(x, height-1, ch, termbox.ColorDefault, termbox.ColorDefault)
			x++
		}

		x++
	}

	first := ui.offsetY
	last := clamp(ui.offsetY+height, 0, len(ui.filterMatches))
	total := len(ui.filterMatches)
	filterString := ""

	if _, ok := ui.filter.(EmptyFilter); !ok {
		filterString = fmt.Sprintf("filter:\"%s\" :: ", ui.filter.String())
	}

	right := fmt.Sprintf("%srows %d-%d of %d", filterString, first, last, total)
	x = len(right)
	for _, ch := range right {
		termbox.SetCell(width-x, height-1, ch, termbox.ColorGreen|termbox.AttrBold, termbox.ColorDefault)
		x--
	}
}

func (ui *UI) writeCell(cell string, x, y, index, pinBound int, fg, bg termbox.Attribute) int {
	col := ui.columns[index]

	if col.Highlight {
		fg = HiliteFg
		bg = HiliteBg
	}

	formatted := cell

	switch col.Display {
	case ColumnDefault:
		width := clamp(col.Width, 0, MaxCellWidth)

		if len(formatted) > width {
			formatted = fmt.Sprintf("%-*s…", width-1, formatted[:width-1])
		} else {
			formatted = fmt.Sprintf("%-*s", width, formatted)
		}
	case ColumnExpanded:
		if len(formatted) < col.Width {
			formatted = fmt.Sprintf("%-*s", col.Width, formatted)
		}
	case ColumnCollapsed:
		formatted = "…"
	case ColumnAligned:
		width := clamp(col.Width, 16, MaxCellWidth)

		if val, err := strconv.ParseFloat(cell, 64); err == nil {
			formatted = fmt.Sprintf("%*.4f", width, val)
		} else {
			formatted = fmt.Sprintf("%*s", width, formatted)
		}
	}

	x = writeStringBounded(x, y, pinBound, fg, bg, formatted)

	// Draw separator if this isn't the last element
	if index != len(ui.columns)-1 {
		x = writeStringBounded(x, y, pinBound, termbox.ColorWhite, termbox.ColorDefault, CellSeparator)
	}

	return x
}

func (ui *UI) writePinned(y int, fg, bg termbox.Attribute, row []string) int {
	// ignore our view offsets
	pinnedBounds := 0

	for i, cell := range row {
		col := ui.columns[i]

		if col.Pinned {
			pinnedBounds = ui.writeCell(cell, pinnedBounds, y, i, -1, fg, bg)
		}
	}

	return pinnedBounds
}

func (ui *UI) writeColumns(x, y int) {
	fg := termbox.ColorGreen | termbox.AttrBold | termbox.AttrUnderline
	bg := termbox.ColorDefault

	colNames := make([]string, len(ui.columns))
	for i, col := range ui.columns {
		colNames[i] = col.Name
	}

	pinBound := ui.writePinned(y, termbox.ColorWhite|termbox.AttrBold, termbox.ColorDefault, colNames)
	x += pinBound

	for i, col := range ui.columns {
		if !col.Pinned {
			x = ui.writeCell(col.Name, x, y, i, pinBound, fg, bg)
		}
	}
}

func (ui *UI) writeRow(x, y int, row []string) {
	fg := termbox.ColorDefault

	if ui.zebraStripe && (ui.offsetY+y)%2 == 0 {
		fg = termbox.ColorMagenta
	}

	pinBound := ui.writePinned(y, termbox.ColorCyan, termbox.ColorDefault, row)
	x += pinBound

	for i, col := range ui.columns {
		if !col.Pinned {
			x = ui.writeCell(row[i], x, y, i, pinBound, fg, termbox.ColorDefault)
		}
	}
}
