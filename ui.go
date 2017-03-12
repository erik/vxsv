package main

import (
	"fmt"
	"strconv"

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
  R               enter ** ROW SELECT MODE **
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
  a               line up decimal points for floats in this column
  [ESC], Ctrl g   return to ** DEFAULT MODE **

FILTER MODE

  [ESC], Ctrl g   clear filter and return to ** DEFAULT MODE **
  Ctrl w          clear filter
  [ENTER]         apply filter and return to default mode

ROW SELECT MODE

  <arrow keys>    select row
  [ENTER]         Pop open expanded row dialog.

`

type ColumnDisplay int

const (
	ColumnDefault = iota
	ColumnCollapsed
	ColumnExpanded
	ColumnAligned
)

type columnOptions struct {
	display   ColumnDisplay
	pinned    bool
	highlight bool
	width     int
}

func (c *columnOptions) toggleDisplay(mode ColumnDisplay) {
	if c.display == mode {
		c.display = ColumnDefault
	} else {
		c.display = mode
	}
}

func (c columnOptions) displayWidth() int {
	switch c.display {
	case ColumnAligned:
		return clamp(c.width, 16, c.width)
	case ColumnCollapsed:
		return 1
	case ColumnExpanded:
		return c.width
	case ColumnDefault:
		return clamp(c.width, 1, MAX_CELL_WIDTH)
	}

	panic("TODO: this is a bug")
}

type UI struct {
	handlers         []ModeHandler
	rowIdx           int // Selection control
	offsetX, offsetY int // Pan control
	filter           Filter
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

// TODO: Write me
func (ui *UI) writeModeLine(left, right string) {

}

func (ui *UI) writeCell(cell string, x, y, index, pinBound int, fg, bg termbox.Attribute) int {
	colOpts := ui.columnOpts[index]

	if colOpts.highlight {
		fg = HILITE_FG
		bg = HILITE_BG
	}

	formatted := cell

	switch colOpts.display {
	case ColumnDefault:
		width := clamp(colOpts.width, 0, MAX_CELL_WIDTH)

		if len(formatted) > width {
			formatted = fmt.Sprintf("%-*s…", width-1, formatted[:width-1])
		} else {
			formatted = fmt.Sprintf("%-*s", width, formatted)
		}
	case ColumnExpanded:
		if len(formatted) < colOpts.width {
			formatted = fmt.Sprintf("%-*s", colOpts.width, formatted)
		}
	case ColumnCollapsed:
		formatted = "…"
	case ColumnAligned:
		width := clamp(colOpts.width, 16, MAX_CELL_WIDTH)

		if val, err := strconv.ParseFloat(cell, 64); err == nil {
			formatted = fmt.Sprintf("%*.4f", width, val)
		} else {
			formatted = fmt.Sprintf("%*s", width, formatted)
		}
	}

	x = writeStringBounded(x, y, pinBound, fg, bg, formatted)

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

func NewUi(data *TabularData) *UI {
	colOpts := make([]columnOptions, len(data.Columns))
	columns := make([]string, len(data.Columns))
	filterMatches := make([]int, len(data.Rows))

	for i, col := range data.Columns {
		columns[i] = col.Name
		colOpts[i] = columnOptions{
			display:   ColumnDefault,
			pinned:    false,
			highlight: false,
			width:     col.Width,
		}

		// Last column should open expanded
		if i == len(data.Columns)-1 {
			colOpts[i].display = ColumnExpanded
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
		filter:        EmptyFilter{},
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
			if ui.filter.Matches(ui.rows[rowIdx]) {
				rows = append(rows, rowIdx)
			}
		}

		ui.filterMatches = rows
	} else {
		rows := make([]int, 0, 100)

		// FIXME: this +ui.offsetY thing feels like a bug
		for i := 0; i+ui.offsetY < len(ui.rows); i += 1 {
			if ui.filter.Matches(ui.rows[i+ui.offsetY]) {
				rows = append(rows, i)
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

	ui.writeColumns(-ui.offsetX, 0)

	for i := 0; i < vh; i += 1 {
		if i+ui.offsetY < len(ui.filterMatches) {
			ui.writeRow(-ui.offsetX, i+1, ui.rows[ui.filterMatches[i+ui.offsetY]])
		} else {
			writeLine(0, i+1, termbox.ColorWhite|termbox.AttrBold, termbox.ColorBlack, "~")
		}
	}

	ui.activeHandler().Repaint()
	termbox.Flush()
}

func (ui *UI) viewSize() (int, int) {
	width, height := termbox.Size()
	pinnedWidth := ui.pinnedWidth()

	return width - pinnedWidth, height - 2
}

func (ui *UI) pinnedWidth() (width int) {
	for _, colOpt := range ui.columnOpts {
		if colOpt.pinned {
			width += colOpt.displayWidth()
			width += len(CELL_SEPARATOR)
		}
	}

	return width
}

func (ui *UI) columnOffset(colIdx int) (offset int, width int) {
	offset = 0

	for i, _ := range ui.columns {
		colOpts := ui.columnOpts[i]

		// Pinned columns should always be visible
		if i == colIdx {
			if colOpts.pinned {
				return 0, colOpts.width
			} else {
				break
			}
		}

		if !colOpts.pinned {
			width = colOpts.displayWidth()
			offset += width
			offset += len(CELL_SEPARATOR)
		}
	}

	return offset, width
}

// TODO: Write me, stop manually scrolling
func (ui *UI) scroll(rows int) {

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
