// ui mode specific handlers

package main

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/montanaflynn/stats"
	"github.com/nsf/termbox-go"
)

type ModeHandler interface {
	HandleKey(ev termbox.Event)
	Repaint()
}

type HandlerDefault struct {
	ui *UI
}

func (h *HandlerDefault) Repaint() {
	ui := h.ui
	_, height := termbox.Size()

	first := ui.offsetY
	last := clamp(ui.offsetY+height, 0, len(ui.filterMatches))
	total := len(ui.filterMatches) - 1
	filter := ""

	if _, ok := ui.filter.(EmptyFilter); !ok {
		filter = fmt.Sprintf("[filter: \"%s\"] ", ui.filter.String())
	}

	line := fmt.Sprintf("%s[rows %d-%d of %d] :%d, %d", filter, first, last, total, ui.offsetX, ui.offsetY)
	writeLine(0, height-1, termbox.ColorDefault, termbox.ColorDefault, line)
}

func (h *HandlerDefault) HandleKey(ev termbox.Event) {
	ui := h.ui
	vw, vh := ui.viewSize()

	maxYOffset := clamp(len(ui.filterMatches)-(vh-2), 0, len(ui.filterMatches)-1)
	lastColumnOffset, _ := ui.columnOffset(len(ui.columns) - 1)
	endOfLine := lastColumnOffset - vw

	// prevent funky scrolling behavior when row is smaller than screen
	if endOfLine < 0 {
		endOfLine = 0
	}

	switch {
	case ev.Key == termbox.KeyCtrlL:
		termbox.Sync()
	case ev.Key == termbox.KeyCtrlA:
		ui.offsetX = 0
	case ev.Key == termbox.KeyCtrlE:
		ui.offsetX = endOfLine
	case ev.Key == termbox.KeyArrowRight:
		ui.offsetX = clamp(ui.offsetX+5, 0, endOfLine)
	case ev.Key == termbox.KeyArrowLeft:
		ui.offsetX = clamp(ui.offsetX-5, 0, endOfLine)
	case ev.Key == termbox.KeyArrowUp:
		ui.offsetY = clamp(ui.offsetY-1, 0, maxYOffset)
	case ev.Key == termbox.KeyArrowDown:
		ui.offsetY = clamp(ui.offsetY+1, 0, maxYOffset)
	case ev.Ch == '/', ev.Key == termbox.KeyCtrlR:
		ui.pushHandler(&HandlerFilter{*h, ui.filter.String()})
		ui.offsetY = 0
	case ev.Key == termbox.KeySpace:
		ui.offsetY = clamp(ui.offsetY+vh, 0, maxYOffset)
	case ev.Ch == 'C':
		ui.pushHandler(NewColumnSelect(h.ui))
		ui.offsetX = 0
	case ev.Ch == 'G':
		ui.offsetY = maxYOffset
	case ev.Ch == 'g':
		ui.offsetY = 0
	case ev.Ch == 'Z':
		ui.zebraStripe = !ui.zebraStripe
	case ev.Ch == 'X':
		for i, _ := range ui.columnOpts {
			if globalExpanded {
				ui.columnOpts[i].display = ColumnExpanded
			} else {
				ui.columnOpts[i].display = ColumnDefault
			}
		}
		globalExpanded = !globalExpanded
	case ev.Ch == '?':
		ui.pushHandler(NewPopup(h.ui, HELP_TEXT))
	case ev.Ch == 'r':
		ui.pushHandler(&HandlerRowSelect{*h, h.ui.offsetY})
	}
}

type HandlerFilter struct {
	HandlerDefault
	filter string
}

func (h *HandlerFilter) Repaint() {
	ui := h.ui
	_, height := termbox.Size()

	line := fmt.Sprintf("FILTER [%d matches]: %s", len(ui.filterMatches), h.filter)
	writeLine(0, height-1, termbox.ColorWhite|termbox.AttrBold, termbox.ColorDefault, line)
	termbox.SetCursor(len(line), height-1)

}

func (h *HandlerFilter) HandleKey(ev termbox.Event) {
	ui := h.ui

	// Ch == 0 implies this was a special key
	if ev.Ch == 0 && ev.Key != termbox.KeySpace {
		if ev.Key == termbox.KeyEsc || ev.Key == termbox.KeyCtrlG {
			ui.popHandler()

			ui.filter = EmptyFilter{}
			ui.filterRows(false)
		} else if ev.Key == termbox.KeyEnter {
			if filter, err := ui.parseFilter(h.filter); err == nil {
				ui.filter = filter
			} else {
				ui.pushHandler(NewPopup(ui, fmt.Sprintf("There was an error in your filter:\n\n%v\n\n%s", err, h.filter)))
				return
			}
			ui.filterRows(false)
			ui.popHandler()
		} else if ev.Key == termbox.KeyDelete || ev.Key == termbox.KeyBackspace ||
			ev.Key == termbox.KeyBackspace2 {
			if sz := len(h.filter); sz > 0 {
				h.filter = h.filter[:sz-1]
			}
		} else if ev.Key == termbox.KeyCtrlW || ev.Key == termbox.KeyCtrlU {
			h.filter = ""
			ui.filterRows(false)
		} else {
			// Fallback to default handling for arrows etc
			// FIXME: is this really the best way to do this in go?
			def := &HandlerDefault{h.ui}
			def.HandleKey(ev)
		}
		return
	}

	if ev.Key == termbox.KeySpace {
		h.filter += " "
	} else {
		h.filter += string(ev.Ch)
	}

	ui.offsetY = 0
	ui.filterRows(true)
}

type HandlerRowSelect struct {
	HandlerDefault
	rowIdx int
}

func (h *HandlerRowSelect) HandleKey(ev termbox.Event) {
	ui := h.ui

	switch ev.Key {
	case termbox.KeyEsc, termbox.KeyCtrlG:
		ui.popHandler()
	case termbox.KeyArrowUp:
		h.rowIdx = clamp(h.rowIdx-1, 0, len(ui.filterMatches))
	case termbox.KeyArrowDown:
		h.rowIdx = clamp(h.rowIdx+1, 0, len(ui.filterMatches))
	case termbox.KeyEnter:
		jsonObj := make(map[string]interface{})

		row := ui.rows[h.rowIdx]
		for i, col := range ui.columns {
			str := row[i]

			if v, err := strconv.ParseInt(str, 10, 64); err == nil {
				jsonObj[col] = v
			} else if v, err := strconv.ParseFloat(str, 64); err == nil {
				jsonObj[col] = v
			} else if v, err := strconv.ParseBool(str); err == nil {
				jsonObj[col] = v
			} else {
				jsonObj[col] = str
			}

		}

		jsonStr, err := json.MarshalIndent(jsonObj, "", "  ")
		// TODO: This should be handled
		if err != nil {
			panic(err)
		}

		ui.pushHandler(NewPopup(ui, string(jsonStr)))
	default:
		def := &HandlerDefault{ui}
		def.HandleKey(ev)
	}

	_, height := ui.viewSize()
	if h.rowIdx-ui.offsetY >= height {
		ui.offsetY = clamp(ui.offsetY+1, 0, len(ui.filterMatches))
	} else if h.rowIdx < ui.offsetY {
		ui.offsetY = h.rowIdx
	}
}

func (h *HandlerRowSelect) Repaint() {
	ui := h.ui
	_, height := termbox.Size()

	termbox.SetCell(0, 1+h.rowIdx-ui.offsetY, ROW_INDICATOR, termbox.ColorRed|termbox.AttrBold, termbox.ColorWhite)

	line := fmt.Sprintf("ROW SELECT (^g quit) [%d of %d]", h.rowIdx, len(ui.filterMatches))
	writeLine(0, height-1, termbox.ColorWhite|termbox.AttrBold, termbox.ColorDefault, line)
}

type HandlerColumnSelect struct {
	HandlerDefault
	column int
}

func NewColumnSelect(ui *UI) *HandlerColumnSelect {
	h := HandlerColumnSelect{
		HandlerDefault: HandlerDefault{ui},
		column:         0,
	}

	h.selectColumn(0)
	return &h
}

// TODO: also adjust x offset
func (h *HandlerColumnSelect) selectColumn(idx int) {
	h.ui.columnOpts[h.column].highlight = false
	h.column = idx

	if h.column >= 0 {
		h.ui.columnOpts[h.column].highlight = true
	}
}

func (h *HandlerColumnSelect) rowSorter(i, j int) bool {
	ui := h.ui

	row1 := ui.rows[ui.filterMatches[i]]
	row2 := ui.rows[ui.filterMatches[j]]

	v1, err1 := strconv.ParseFloat(row1[h.column], 32)
	v2, err2 := strconv.ParseFloat(row2[h.column], 32)

	if err1 == nil && err2 == nil {
		return v1 < v2
	}

	return row1[h.column] < row2[h.column]
}

func (h *HandlerColumnSelect) Repaint() {
	ui := h.ui
	_, height := termbox.Size()

	line := fmt.Sprintf("COLUMN SELECT (^g quit) [%s] %d", ui.columns[h.column], h.column)
	writeLine(0, height-1, termbox.ColorWhite|termbox.AttrBold, termbox.ColorDefault, line)
}

// FIXME: When transitioning out of ColumnSelect mode, we leave a
// FIXME: column highlighted.
func (h *HandlerColumnSelect) HandleKey(ev termbox.Event) {
	ui := h.ui
	savedColumn := -1
	colOpt := &ui.columnOpts[h.column]

	switch {
	case ev.Key == termbox.KeyCtrlA:
		h.selectColumn(ui.findFirstColumn())
	case ev.Key == termbox.KeyCtrlE:
		h.selectColumn(len(ui.columns) - 1)
	case ev.Key == termbox.KeyArrowRight:
		next := ui.findNextColumn(h.column, 1)
		h.selectColumn(clamp(next, 0, len(ui.columns)-1))
	case ev.Key == termbox.KeyArrowLeft:
		next := ui.findNextColumn(h.column, -1)
		h.selectColumn(clamp(next, 0, len(ui.columns)-1))
	case ev.Ch == '<':
		sort.SliceStable(ui.filterMatches, func(i, j int) bool {
			return h.rowSorter(i, j)
		})
	case ev.Ch == '>':
		sort.SliceStable(ui.filterMatches, func(i, j int) bool {
			return h.rowSorter(j, i)
		})
	case ev.Ch == 'w':
		colOpt.toggleDisplay(ColumnCollapsed)
	case ev.Ch == 'x':
		colOpt.toggleDisplay(ColumnExpanded)
	case ev.Ch == 'a':
		colOpt.toggleDisplay(ColumnAligned)
	case ev.Ch == '.':
		colOpt.pinned = !colOpt.pinned

		if colOpt.pinned {
			colOpt.display = ColumnDefault
		}
	case ev.Ch == 's':
		var (
			min, max, stdev    float64
			mean, median, mode float64
			modes              []float64
			sum, variance      float64
			p90, p95, p99      float64
			quartiles          stats.Quartiles
			err                error
			text               string
		)

		data := make(stats.Float64Data, 0, len(ui.filterMatches)+1)
		for _, rowIdx := range ui.filterMatches {
			row := ui.rows[rowIdx]
			trimmed := strings.TrimSpace(row[h.column])
			if val, err := strconv.ParseFloat(trimmed, 64); err == nil {
				data = append(data, val)
			}
		}

		if len(data) == 0 {
			data = []float64{0.0, 0.0, 0.0, 0.0}
		}

		// The joy of go
		if min, err = data.Min(); err != nil {
			goto error
		} else if max, err = data.Max(); err != nil {
			goto error
		} else if mean, err = data.Mean(); err != nil {
			goto error
		} else if median, err = data.Median(); err != nil {
			goto error
		} else if modes, err = data.Mode(); err != nil {
			goto error
		} else if stdev, err = data.StandardDeviation(); err != nil {
			goto error
		} else if sum, err = data.Sum(); err != nil {
			goto error
		} else if p90, err = data.Percentile(90); err != nil {
			goto error
		} else if p95, err = data.Percentile(95); err != nil {
			goto error
		} else if p99, err = data.Percentile(99); err != nil {
			goto error
		} else if variance, err = data.Variance(); err != nil {
			goto error
		} else if quartiles, err = stats.Quartile(data); err != nil {
			goto error
		}

		if len(modes) > 0 {
			mode = modes[0]
		} else {
			mode = math.NaN()
		}

		text = fmt.Sprintf(`
  SUMMARY STATISTICS [ %s ]
  ------------------
  rows visible: %d (of %d)
  numeric rows: %d

  min: %15.4f      mean:   %15.4f
  max: %15.4f      median: %15.4f
  sum: %15.4f      mode:   %15.4f

  var: %15.4f      std:    %15.4f

  p90: %15.4f      p25:    %15.4f
  p95: %15.4f      p50:    %15.4f
  p99: %15.4f      p75:    %15.4f`,
			ui.columns[h.column], len(ui.filterMatches), len(ui.rows), len(data),
			min, mean, max, median, sum, mode, variance, stdev,
			p90, quartiles.Q1, p95, quartiles.Q2, p99, quartiles.Q3)

		ui.pushHandler(NewPopup(ui, text))

		break
		// TODO: handle rror for real
	error:
		fmt.Printf("%v\n", err)
		panic(err)
	case ev.Key == termbox.KeyCtrlG, ev.Key == termbox.KeyEsc:
		savedColumn = h.column
		h.selectColumn(-1)
		ui.popHandler()
	default:
		// FIXME: ditto, is this the best way to do this?
		def := &HandlerDefault{h.ui}
		def.HandleKey(ev)
	}

	// find if we've gone off screen and readjust
	// TODO: this bit is buggy
	cursorPosition := 0
	for i, _ := range ui.columns {
		colOpts := ui.columnOpts[i]

		if i == h.column || i == savedColumn {
			break
		}

		switch colOpts.display {
		case ColumnCollapsed:
			cursorPosition += 1
		case ColumnExpanded:
			cursorPosition += colOpts.width
		case ColumnDefault:
			cursorPosition += clamp(colOpts.width, 0, MAX_CELL_WIDTH)
		}

		cursorPosition += len(CELL_SEPARATOR)
	}

	width, _ := ui.viewSize()
	if cursorPosition > width-ui.offsetX || cursorPosition < -ui.offsetX {
		ui.offsetX = -cursorPosition
	}
}

type HandlerPopup struct {
	HandlerDefault

	content          []string
	offsetX, offsetY int
}

func NewPopup(ui *UI, content string) *HandlerPopup {
	return &HandlerPopup{
		HandlerDefault: HandlerDefault{ui},
		content:        strings.Split(content, "\n"),
	}
}

func (h *HandlerPopup) size() (int, int) {
	width, height := termbox.Size()

	popupW := clamp(120, 50, width-15)
	popupH := clamp(len(h.content), 10, height-5)

	return popupW, popupH
}

func (h *HandlerPopup) Repaint() {
	width, height := termbox.Size()
	popupW, popupH := h.size()

	x := width/2 - popupW/2
	y := height/2 - popupH/2

	borders := [][]string{
		[]string{"┌─", "─┐"},
		[]string{"│ ", " │"},
		[]string{"└─", "─┘"},
	}

	for i := -1; i <= popupH; i += 1 {
		var border []string
		var content string

		if i == -1 {
			border = borders[0]
			content = strings.Repeat("─", popupW)
		} else if i < popupH {
			border = borders[1]
			if i+h.offsetY < len(h.content) {
				content = h.content[i+h.offsetY]
				if len(content) >= popupW {
					content = content[0:popupW]
				}
			} else {
				content = " "
			}
		} else {
			border = borders[2]
			content = strings.Repeat("─", popupW)
		}

		line := fmt.Sprintf("%s%-*s%s", border[0], popupW, content, border[1])
		writeString(x, y+i, termbox.ColorWhite, termbox.ColorDefault, line)
	}

	line := "POPUP (^g quit)"
	writeLine(0, height-1, termbox.ColorWhite|termbox.AttrBold, termbox.ColorDefault, line)

}

func (h *HandlerPopup) HandleKey(ev termbox.Event) {
	if ev.Key == termbox.KeyEsc || ev.Key == termbox.KeyCtrlG || ev.Ch == 'q' {
		h.ui.popHandler()
	}

	_, maxScroll := h.size()
	if maxScroll >= len(h.content) {
		maxScroll = 0
	}

	switch ev.Key {
	case termbox.KeyArrowLeft:
		h.offsetX = clamp(h.offsetX-5, 0, 9999)
	case termbox.KeyArrowRight:
		h.offsetX = clamp(h.offsetX+5, 0, 9999)
	case termbox.KeyArrowUp:
		h.offsetY = clamp(h.offsetY-1, 0, maxScroll)
	case termbox.KeyArrowDown:
		h.offsetY = clamp(h.offsetY+1, 0, maxScroll)
	}
}
