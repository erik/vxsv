package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/nsf/termbox-go"
)

const MAX_CELL_WIDTH = 20
const CELL_SEPARATOR = " │ "
const HILITE_FG = termbox.ColorBlack | termbox.AttrBold
const HILITE_BG = termbox.ColorWhite

type inputMode int

const (
	ModeDefault = iota
	ModeFilter
	ModeColumnSelect
	ModeRowSelect
)

// It is so dumb that go doesn't have this
func clamp(val, lo, hi int) int {
	if val <= lo {
		return lo
	} else if val >= hi {
		return hi
	}

	return val
}

var pinnedBounds = 0

func writeString(x, y int, fg, bg termbox.Attribute, msg string) int {
	for _, c := range msg {
		if x >= pinnedBounds {
			termbox.SetCell(x, y, c, fg, bg)
		}
		x += 1
	}
	return x
}

// Fill entire line of screen
func writeLine(x, y int, fg, bg termbox.Attribute, line string) {
	width, _ := termbox.Size()
	for _, c := range line {
		termbox.SetCell(x, y, c, fg, bg)
		x += 1
	}
	for i := x; i < width; i += 1 {
		termbox.SetCell(x+i, y, ' ', fg, bg)
	}
}

var cellFmtString = "%-" + strconv.Itoa(MAX_CELL_WIDTH) + "s"

func (ui *UI) writeCell(cell string, x, y, index int, fg, bg termbox.Attribute) int {
	colOpts := ui.columnOpts[index]
	lastCol := index == len(ui.columnOpts)-1

	if index == ui.colIdx && ui.mode == ModeColumnSelect {
		fg = HILITE_FG
		bg = HILITE_BG
	}

	if colOpts.collapsed {
		x = writeString(x, y, fg, bg, "…")
	} else if !colOpts.expanded && len(cell) < MAX_CELL_WIDTH {
		padded := fmt.Sprintf(cellFmtString, cell)
		x = writeString(x, y, fg, bg, padded)
	} else if !colOpts.expanded && !lastCol {
		width := clamp(len(cell)-1, 0, MAX_CELL_WIDTH-1)
		x = writeString(x, y, fg, bg, cell[:width])
		x = writeString(x, y, fg, bg, "…")
	} else {
		fmtString := "%-" + strconv.Itoa(colOpts.width) + "s"
		writeString(x, y, fg, bg, fmt.Sprintf(fmtString, cell))
		x += colOpts.width
	}

	// Draw separator if this isn't the last element
	if index != len(ui.columns)-1 {
		x = writeString(x, y, termbox.ColorRed, termbox.ColorDefault, CELL_SEPARATOR)
	}

	return x
}

func (ui *UI) writePinned(y int, fg, bg termbox.Attribute, row []string) int {
	// ignore our view offsets
	pinnedBounds = 0

	for i, cell := range row {
		colOpts := ui.columnOpts[i]

		if colOpts.pinned {
			pinnedBounds = ui.writeCell(cell, pinnedBounds, y, i, fg, bg)
		}
	}

	return pinnedBounds
}

func (ui *UI) writeColumns(x, y int) {
	var fg, bg termbox.Attribute

	x += ui.writePinned(y, termbox.ColorWhite, termbox.ColorDefault, ui.columns)

	for i, col := range ui.columns {
		colOpts := ui.columnOpts[i]

		fg = termbox.ColorBlack | termbox.AttrBold
		bg = termbox.ColorWhite

		if !colOpts.pinned {
			x = ui.writeCell(col, x, y, i, fg, bg)
		}
	}

	endOfLinePosition = x
}

func (ui *UI) writeRow(x, y int, row []string) {
	fg := termbox.ColorDefault

	if ui.zebraStripe && y%2 == 0 {
		fg = termbox.ColorMagenta
	}

	x += ui.writePinned(y, termbox.ColorCyan, termbox.ColorBlack, row)

	for i, _ := range ui.columns {
		colOpts := ui.columnOpts[i]

		if !colOpts.pinned {
			x = ui.writeCell(row[i], x, y, i, fg, termbox.ColorDefault)
		}
	}
}

type columnOptions struct {
	expanded  bool
	collapsed bool
	pinned    bool
	width     int
}

type UI struct {
	mode             inputMode
	rowIdx, colIdx   int // Selection control
	offsetX, offsetY int // Pan control
	filterString     string
	filterMatches    []int
	zebraStripe      bool
	columnOpts       []columnOptions
	columns          []string
	rows             [][]string
	width            int
}

func NewUi(data TabularData) UI {
	colOpts := make([]columnOptions, len(data.Columns))
	columns := make([]string, len(data.Columns))
	filterMatches := make([]int, len(data.Rows))

	for i, col := range data.Columns {
		columns[i] = col.Name
		colOpts[i] = columnOptions{
			expanded:  col.Width < MAX_CELL_WIDTH,
			collapsed: false,
			pinned:    false,
			width:     col.Width,
		}
	}

	for i, _ := range data.Rows {
		filterMatches[i] = i
	}

	return UI{
		offsetX:       0,
		offsetY:       0,
		mode:          ModeDefault,
		colIdx:        -1,
		columnOpts:    colOpts,
		rows:          data.Rows,
		columns:       columns,
		width:         data.Width,
		zebraStripe:   false,
		filterString:  "",
		filterMatches: filterMatches,
	}
}

func (ui *UI) Init() error {
	if err := termbox.Init(); err != nil {
		return err
	}

	termbox.SetInputMode(termbox.InputEsc)

	return nil
}

func (ui *UI) Loop() {
	defer termbox.Close()

	ui.repaint()

eventloop:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			if ev.Key == termbox.KeyCtrlC {
				break eventloop
			}

			switch ui.mode {
			case ModeFilter:
				ui.handleKeyFilter(ev)
			case ModeColumnSelect:
				ui.handleKeyColumnSelect(ev)
			default:
				ui.handleKeyDefault(ev)
			}
		}

		ui.repaint()
	}
}

// Return indices of rows to display
func (ui *UI) filterRows(narrowing bool) {

	// If we are adding a character to the filter, no need to start from
	// scratch, this will be a strict subset of our current filter.
	if narrowing {
		rows := make([]int, 0, len(ui.filterMatches))

		for _, rowIdx := range ui.filterMatches {
			for _, col := range ui.rows[rowIdx] {
				if strings.Contains(col, ui.filterString) {
					rows = append(rows, rowIdx)
					break
				}
			}
		}

		ui.filterMatches = rows
	} else {
		rows := make([]int, 0, 100)

		for i := 0; i+ui.offsetY < len(ui.rows); i += 1 {
			for _, col := range ui.rows[i+ui.offsetY] {
				if ui.filterString == "" || strings.Contains(col, ui.filterString) {
					rows = append(rows, i)
					break
				}
			}
		}

		ui.filterMatches = rows
	}
}

func (ui *UI) repaint() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	termbox.HideCursor()
	_, height := termbox.Size()

	const coldef = termbox.ColorDefault

	ui.writeColumns(ui.offsetX+0, 0)

	for i := 0; i < height-2; i += 1 {
		if i+ui.offsetY < len(ui.filterMatches) {
			ui.writeRow(ui.offsetX+0, i+1, ui.rows[ui.filterMatches[i+ui.offsetY]])
		} else {
			writeLine(0, i+1, termbox.ColorWhite|termbox.AttrBold, termbox.ColorBlack, "~")
		}
	}

	switch ui.mode {
	case ModeFilter:
		line := fmt.Sprintf("FILTER [%d matches]: %s", len(ui.filterMatches), ui.filterString)
		writeLine(0, height-1, termbox.ColorWhite|termbox.AttrBold, termbox.ColorDefault, line)
		termbox.SetCursor(len(line), height-1)
	case ModeColumnSelect:
		line := "COLUMN SELECT (^g quit) [" + ui.columns[ui.colIdx] + "]"
		writeLine(0, height-1, termbox.ColorWhite|termbox.AttrBold, termbox.ColorDefault, line)
	default:
		first := ui.offsetY
		last := clamp(ui.offsetY+height, 0, len(ui.filterMatches))
		total := len(ui.filterMatches) - 1
		filter := ""

		if ui.filterString != "" {
			filter = fmt.Sprintf("[filter: \"%s\"] ", ui.filterString)
		}

		line := fmt.Sprintf("%s[rows %d-%d of %d] :", filter, first, last, total)
		writeLine(0, height-1, termbox.ColorDefault, termbox.ColorDefault, line)
	}

	termbox.Flush()
}

func (ui *UI) handleKeyFilter(ev termbox.Event) {
	// Ch == 0 implies this was a special key
	if ev.Ch == 0 && ev.Key != termbox.KeySpace {
		if ev.Key == termbox.KeyEsc || ev.Key == termbox.KeyCtrlG || ev.Key == termbox.KeyEnter {
			ui.mode = ModeDefault
		} else if ev.Key == termbox.KeyDelete || ev.Key == termbox.KeyBackspace ||
			ev.Key == termbox.KeyBackspace2 {
			if sz := len(ui.filterString); sz > 0 {
				ui.filterString = ui.filterString[:sz-1]
				ui.filterRows(false)
			}
		} else if ev.Key == termbox.KeyCtrlW || ev.Key == termbox.KeyCtrlU {
			ui.filterString = ""
			ui.filterRows(false)
		} else {
			// Fallback to default handling for arrows etc
			ui.handleKeyDefault(ev)
		}
		return
	}

	if ev.Key == termbox.KeySpace {
		ui.filterString += " "
	} else {
		ui.filterString += string(ev.Ch)
	}

	ui.offsetY = 0
	ui.filterRows(true)
}

var globalExpanded = false
var endOfLinePosition = 0

func (ui *UI) findNextColumn(current, direction int) int {
	isPinned := ui.columnOpts[current].pinned

	// if pinned, find the next pinned col, or vice versa for unpinned
	for i := current + direction; i >= 0 && i < len(ui.columns); i += direction {
		if ui.columnOpts[i].pinned == isPinned {
			return i
		}
	}

	// Don't fall off the end (would cause confusing behavior)
	if isPinned && direction < 0 || !isPinned && direction > 0 {
		return current
	}

	// we're crossing the pinned/unpinned boundary
	i := 0
	if direction < 0 {
		i = len(ui.columns) - 1
	}
	for ; i >= 0 && i < len(ui.columns); i += direction {
		if (isPinned && !ui.columnOpts[i].pinned) || (!isPinned && ui.columnOpts[i].pinned) {
			return i
		}
	}

	return current
}

func (ui *UI) rowSorter(i, j int) bool {
	row1 := ui.rows[ui.filterMatches[i]]
	row2 := ui.rows[ui.filterMatches[j]]

	v1, err1 := strconv.ParseFloat(row1[ui.colIdx], 32)
	v2, err2 := strconv.ParseFloat(row2[ui.colIdx], 32)

	if err1 == nil && err2 == nil {
		return v1 < v2
	}

	return row1[ui.colIdx] < row2[ui.colIdx]
}

func (ui *UI) handleKeyColumnSelect(ev termbox.Event) {
	switch {
	case ev.Key == termbox.KeyArrowRight:
		next := ui.findNextColumn(ui.colIdx, 1)
		ui.colIdx = clamp(next, 0, len(ui.columns)-1)

	case ev.Key == termbox.KeyArrowLeft:
		next := ui.findNextColumn(ui.colIdx, -1)
		ui.colIdx = clamp(next, 0, len(ui.columns)-1)
	case ev.Ch == '<':
		sort.SliceStable(ui.filterMatches, func(i, j int) bool {
			return ui.rowSorter(i, j)
		})
	case ev.Ch == '>':
		sort.SliceStable(ui.filterMatches, func(i, j int) bool {
			return ui.rowSorter(j, i)
		})
	case ev.Ch == 'w':
		ui.columnOpts[ui.colIdx].collapsed = !ui.columnOpts[ui.colIdx].collapsed
	case ev.Ch == 'x':
		ui.columnOpts[ui.colIdx].expanded = !ui.columnOpts[ui.colIdx].expanded
		if ui.columnOpts[ui.colIdx].expanded {
			ui.columnOpts[ui.colIdx].collapsed = false
		}
	case ev.Ch == '.':
		ui.columnOpts[ui.colIdx].pinned = !ui.columnOpts[ui.colIdx].pinned

		if ui.columnOpts[ui.colIdx].pinned {
			ui.columnOpts[ui.colIdx].collapsed = false
		}

	case ev.Key == termbox.KeyCtrlG, ev.Key == termbox.KeyEsc:
		ui.mode = ModeDefault
	default:
		ui.handleKeyDefault(ev)
	}

	// find if we've gone off screen and readjust
	// TODO: this bit is buggy
	cursorPosition := 0
	for i, _ := range ui.columns {
		colOpts := ui.columnOpts[i]

		if i == ui.colIdx {
			break
		}
		//cursorPosition += 3
		if !colOpts.collapsed {
			cursorPosition += colOpts.width
		}
	}

	width, _ := termbox.Size()
	if cursorPosition > width-ui.offsetX || cursorPosition < -ui.offsetX {
		ui.offsetX = -cursorPosition
	}
}

func (ui *UI) handleKeyDefault(ev termbox.Event) {
	// FIXME: this is buggy
	w, h := termbox.Size()
	maxYOffset := clamp(len(ui.filterMatches)-(h-5), 0, len(ui.filterMatches)-1)

	switch {
	case ev.Key == termbox.KeyCtrlA:
		ui.offsetX = 0
	case ev.Key == termbox.KeyCtrlE:
		ui.offsetX = -endOfLinePosition + w
	case ev.Key == termbox.KeyArrowRight:
		ui.offsetX = clamp(ui.offsetX-5, -endOfLinePosition, 0)
	case ev.Key == termbox.KeyArrowLeft:
		ui.offsetX = clamp(ui.offsetX+5, -endOfLinePosition, 0)
	case ev.Key == termbox.KeyArrowUp:
		ui.offsetY = clamp(ui.offsetY-1, 0, maxYOffset)
	case ev.Key == termbox.KeyArrowDown:
		ui.offsetY = clamp(ui.offsetY+1, 0, maxYOffset)
	case ev.Ch == '/', ev.Key == termbox.KeyCtrlR:
		ui.mode = ModeFilter
		ui.offsetY = 0
	case ev.Ch == 'C':
		ui.mode = ModeColumnSelect
		ui.offsetX = 0
		ui.colIdx = 0
	case ev.Ch == 'G':
		ui.offsetY = maxYOffset
	case ev.Ch == 'g':
		ui.offsetY = 0
	case ev.Ch == 'Z':
		ui.zebraStripe = !ui.zebraStripe
	case ev.Ch == 'X':
		for i, _ := range ui.columnOpts {
			ui.columnOpts[i].expanded = !globalExpanded
			// FIXME: Possibly not the best behavior
			ui.columnOpts[i].collapsed = false
		}
		globalExpanded = !globalExpanded

	case ui.mode == ModeDefault && ev.Ch == 'q':
		panic("TODO: real exit")
	}
}
