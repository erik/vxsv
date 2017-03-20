package vxsv

import (
	"fmt"
	"strconv"

	"github.com/nsf/termbox-go"
)

const MaxCellWidth = 20
const CellSeparator = " │ "
const RowIndicator = '»'

const HiliteFg = termbox.ColorBlack | termbox.AttrBold
const HiliteBg = termbox.ColorWhite

const HelpText = `Key Bindings:

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

  Filter strings can take two forms:

    1. Column filter: "column_name CMP value"
       * CMP is one of (==, !=, <, <=, >, >=)
       * Display rows where the given column's value for the
         row makes the comparison evaluate to true.
    2. Row filter: "filter_string"
       * Display rows where any column in the row matches the
         filter string.

  [ESC], Ctrl g   clear filter and return to ** DEFAULT MODE **
  Ctrl w          clear filter
  [ENTER]         apply filter and return to default mode

ROW SELECT MODE

  <arrow keys>    select row
  [ENTER]         Pop open expanded row dialog.
`

type Column struct {
	Name  string
	Width int

	// Display options
	Display   ColumnDisplay
	Pinned    bool
	Highlight bool
}

type TabularData struct {
	Columns []Column
	Rows    [][]string
}

type ColumnDisplay int

const (
	ColumnDefault = iota
	ColumnCollapsed
	ColumnExpanded
	ColumnAligned
)

func (c *Column) toggleDisplay(mode ColumnDisplay) {
	if c.Display == mode {
		c.Display = ColumnDefault
	} else {
		c.Display = mode
	}
}

func (c Column) displayWidth() int {
	switch c.Display {
	case ColumnAligned:
		return clamp(c.Width, 16, c.Width)
	case ColumnCollapsed:
		return 1
	case ColumnExpanded:
		return c.Width
	case ColumnDefault:
		return clamp(c.Width, 1, MaxCellWidth)
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
	columns          []Column
	rows             [][]string
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

	var x int = 0
	for _, ch := range mode {
		termbox.SetCell(x, height-1, ch, termbox.ColorWhite|termbox.AttrBold, termbox.ColorDefault)
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
		termbox.SetCell(width-x, height-1, ch, termbox.ColorGreen|termbox.AttrBold, termbox.ColorBlack)
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
	fg := termbox.ColorBlack | termbox.AttrBold
	bg := termbox.ColorWhite

	colNames := make([]string, len(ui.columns))
	for i, col := range ui.columns {
		colNames[i] = col.Name
	}

	pinBound := ui.writePinned(y, termbox.ColorWhite, termbox.ColorDefault, colNames)
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

	pinBound := ui.writePinned(y, termbox.ColorCyan, termbox.ColorBlack, row)
	x += pinBound

	for i, col := range ui.columns {
		if !col.Pinned {
			x = ui.writeCell(row[i], x, y, i, pinBound, fg, termbox.ColorDefault)
		}
	}
}

func NewUI(data *TabularData) *UI {
	filterMatches := make([]int, len(data.Rows))

	for i, col := range data.Columns {
		col.Display = ColumnDefault
		col.Pinned = false
		col.Highlight = false

		// Last column should open expanded
		if i == len(data.Columns)-1 {
			col.Display = ColumnExpanded
		}
	}

	for i := range data.Rows {
		filterMatches[i] = i
	}

	ui := &UI{
		offsetX:       0,
		offsetY:       0,
		rows:          data.Rows,
		columns:       data.Columns,
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
		for i := 0; i+ui.offsetY < len(ui.rows); i++ {
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

	for i := 0; i < vh; i++ {
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
	for _, col := range ui.columns {
		if col.Pinned {
			width += col.displayWidth()
			width += len(CellSeparator)
		}
	}

	return width
}

func (ui *UI) columnOffset(colIdx int) (offset int, width int) {
	offset = 0

	for i, col := range ui.columns {
		// Pinned columns should always be visible
		if i == colIdx {
			if !col.Pinned {
				break
			}

			return 0, col.Width
		}

		if !col.Pinned {
			width = col.displayWidth()
			offset += width
			offset += len(CellSeparator)
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
	for i, col := range ui.columns {
		if col.Pinned {
			return i
		}
	}

	// If there are no pinned columns, just return first
	return 0
}

func (ui *UI) findNextColumn(current, direction int) int {
	isPinned := ui.columns[current].Pinned

	// if pinned, find the next pinned col, or vice versa for unpinned
	for i := current + direction; i >= 0 && i < len(ui.columns); i += direction {
		if ui.columns[i].Pinned == isPinned {
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
		if (isPinned && !ui.columns[i].Pinned) || (!isPinned && ui.columns[i].Pinned) {
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
