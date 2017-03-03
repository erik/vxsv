package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nsf/termbox-go"
)

const MAX_CELL_WIDTH = 20
const CELL_SEPARATOR = " │ "
const ROW_INDICATOR = '»'

const HILITE_FG = termbox.ColorBlack | termbox.AttrBold
const HILITE_BG = termbox.ColorWhite

const HELP_TEXT = `Key Bindings:

vxsv is a modal viewer, meaning that actions are only valid in certain
contexts.

DEFAULT MODE

  Ctrl l          refresh screen
  Ctrl a          pan to beginning of line
  Ctrl e          pan to end of line
  <arrow keys>    scroll / pan control
  Ctrl r, /       enter ** FILTER MODE **
  [ENTER]         pop open dialog showing row in detail
  [SPACE]         scroll down one screen
  C               enter ** COLUMN SELECT MODE **
  G               scroll to bottom
  g               scroll to top
  Z               toggle zebra stripes
  X               toggle expanding all columns
  ?               show this help dialog
  Ctrl c, q       exit

COLUMN SELECT MODE

  <arrow keys>    select column
  Ctrl a          select first column
  Ctrl e          select last column
  <               sort by column, ascending
  >               sort by column, descending
  w               toggle collapsing this column
  x               toggle expanding this column
  .               toggle pinning column
  s               show summary statistics for this column
  [ESC], Ctrl g   return to ** DEFAULT MODE **

FILTER MODE

  [ESC], Ctrl g   clear filter and return to ** DEFAULT MODE **
  Ctrl w          clear filter
  [ENTER]         apply filter and return to default mode
`

type columnOptions struct {
	expanded  bool
	collapsed bool
	pinned    bool
	highlight bool
	width     int
}

type filter interface {
	matches(row []string) bool
}

type rowFilter struct {
	filter        string
	caseSensitive bool
}

type columnFilter struct {
	filter        string
	caseSensitive bool
	colIdx        int
}

func (f rowFilter) matches(row []string) bool {
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

func (f columnFilter) matches(row []string) bool {
	if f.caseSensitive {
		return row[f.colIdx] == f.filter
	}

	lowerFilter := strings.ToLower(f.filter)
	lowerCol := strings.ToLower(row[f.colIdx])

	return strings.Contains(lowerCol, lowerFilter)
}

type UI struct {
	handlers         []ModeHandler
	rowIdx           int // Selection control
	offsetX, offsetY int // Pan control
	filters          []filter
	filterString     string
	filterMatches    []int
	zebraStripe      bool
	columnOpts       []columnOptions
	columns          []string
	rows             [][]string
	width            int
}

// It is so dumb that go doesn't have this
func clamp(val, lo, hi int) int {
	if val <= lo {
		return lo
	} else if val >= hi {
		return hi
	}

	return val
}

func writeStringBounded(x, y, bound int, fg, bg termbox.Attribute, msg string) int {
	for _, c := range msg {
		if x >= bound {
			termbox.SetCell(x, y, c, fg, bg)
		}
		x += 1
	}
	return x
}

func writeString(x, y int, fg, bg termbox.Attribute, msg string) int {
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
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

func (ui *UI) writeCell(cell string, x, y, index, pinBound int, fg, bg termbox.Attribute) int {
	colOpts := ui.columnOpts[index]
	lastCol := index == len(ui.columnOpts)-1

	if colOpts.highlight {
		fg = HILITE_FG
		bg = HILITE_BG
	}

	if colOpts.collapsed {
		x = writeStringBounded(x, y, pinBound, fg, bg, "…")
	} else if !colOpts.expanded && len(cell) < MAX_CELL_WIDTH {
		padded := fmt.Sprintf(cellFmtString, cell)
		x = writeStringBounded(x, y, pinBound, fg, bg, padded)
	} else if !colOpts.expanded && !lastCol {
		width := clamp(len(cell)-1, 0, MAX_CELL_WIDTH-1)
		x = writeStringBounded(x, y, pinBound, fg, bg, cell[:width])
		x = writeStringBounded(x, y, pinBound, fg, bg, "…")
	} else {
		fmtString := "%-" + strconv.Itoa(colOpts.width) + "s"
		writeStringBounded(x, y, pinBound, fg, bg, fmt.Sprintf(fmtString, cell))
		x += colOpts.width
	}

	// Draw separator if this isn't the last element
	if index != len(ui.columns)-1 {
		x = writeStringBounded(x, y, pinBound, termbox.ColorWhite, termbox.ColorDefault, CELL_SEPARATOR)
	}

	return x
}

func (ui *UI) writePinned(y int, fg, bg termbox.Attribute, row []string) int {
	// ignore our view offsets
	pinnedBounds := 0

	for i, cell := range row {
		colOpts := ui.columnOpts[i]

		if colOpts.pinned {
			pinnedBounds = ui.writeCell(cell, pinnedBounds, y, i, -1, fg, bg)
		}
	}

	return pinnedBounds
}

func (ui *UI) writeColumns(x, y int) {
	fg := termbox.ColorBlack | termbox.AttrBold
	bg := termbox.ColorWhite

	pinBound := ui.writePinned(y, termbox.ColorWhite, termbox.ColorDefault, ui.columns)
	x += pinBound

	for i, col := range ui.columns {
		colOpts := ui.columnOpts[i]

		if !colOpts.pinned {
			x = ui.writeCell(col, x, y, i, pinBound, fg, bg)
		}
	}
}

func (ui *UI) writeRow(x, y int, row []string) {
	fg := termbox.ColorDefault

	if ui.zebraStripe && (ui.offsetY+y)%2 == 0 {
		fg = termbox.ColorMagenta
	}

	pinBound := ui.writePinned(y, termbox.ColorCyan, termbox.ColorBlack, row)
	x += pinBound

	for i, _ := range ui.columns {
		colOpts := ui.columnOpts[i]

		if !colOpts.pinned {
			x = ui.writeCell(row[i], x, y, i, pinBound, fg, termbox.ColorDefault)
		}
	}
}

func NewUi(data TabularData) *UI {
	colOpts := make([]columnOptions, len(data.Columns))
	columns := make([]string, len(data.Columns))
	filterMatches := make([]int, len(data.Rows))

	for i, col := range data.Columns {
		columns[i] = col.Name
		colOpts[i] = columnOptions{
			expanded:  col.Width < MAX_CELL_WIDTH,
			collapsed: false,
			pinned:    false,
			highlight: false,
			width:     col.Width,
		}
	}

	for i, _ := range data.Rows {
		filterMatches[i] = i
	}

	ui := &UI{
		offsetX:       0,
		offsetY:       0,
		columnOpts:    colOpts,
		rows:          data.Rows,
		columns:       columns,
		width:         data.Width,
		zebraStripe:   false,
		filters:       []filter{},
		filterString:  "",
		filterMatches: filterMatches,
	}

	ui.switchToDefault()

	return ui
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

			ui.activeHandler().HandleKey(ev)
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
	_, vh := ui.viewSize()

	const coldef = termbox.ColorDefault

	ui.writeColumns(ui.offsetX, 0)

	for i := 0; i < vh; i += 1 {
		if i+ui.offsetY < len(ui.filterMatches) {
			ui.writeRow(ui.offsetX, i+1, ui.rows[ui.filterMatches[i+ui.offsetY]])
		} else {
			writeLine(0, i+1, termbox.ColorWhite|termbox.AttrBold, termbox.ColorBlack, "~")
		}
	}

	ui.activeHandler().Repaint()
	termbox.Flush()
}

func (ui *UI) viewSize() (int, int) {
	width, height := termbox.Size()
	return width, height - 2
}

func (ui *UI) endOfLine() int {
	x := 0

	for i, _ := range ui.columns {
		colOpts := ui.columnOpts[i]

		if !colOpts.pinned {
			if colOpts.collapsed {
				x += 1
			} else if colOpts.expanded {
				x += colOpts.width
			} else {
				x += clamp(colOpts.width, 1, MAX_CELL_WIDTH)
			}

			x += len(CELL_SEPARATOR)
		}
	}

	return x
}

// TODO: Write me, stop manually scrolling
func (ui *UI) scroll(rows int) {

}

// TODO: Write me, stop manually panning
func (ui *UI) panTo(col int) {
}

var globalExpanded = false

// Find the first visually displayed column
func (ui *UI) findFirstColumn() int {
	for i, col := range ui.columnOpts {
		if col.pinned {
			return i
		}
	}

	// If there are no pinned columns, just return first
	return 0
}

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

func (ui *UI) activeHandler() ModeHandler {
	if len(ui.handlers) == 0 {
		ui.switchToDefault()
	}
	return ui.handlers[len(ui.handlers)-1]
}

func (ui *UI) switchToDefault() {
	ui.handlers = []ModeHandler{&HandlerDefault{ui}}
}

func (ui *UI) pushHandler(h ModeHandler) {
	ui.handlers = append(ui.handlers, h)
}

func (ui *UI) popHandler() {
	if len(ui.handlers) > 1 {
		ui.handlers = ui.handlers[:len(ui.handlers)-1]
	} else {
		ui.switchToDefault()
	}
}
