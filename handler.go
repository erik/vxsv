// ui mode specific handlers

package main

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"fmt"
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

	if ui.filterString != "" {
		filter = fmt.Sprintf("[filter: \"%s\"] ", ui.filterString)
	}

	line := fmt.Sprintf("%s[rows %d-%d of %d] :%d, %d", filter, first, last, total, ui.offsetX, ui.offsetY)
	writeLine(0, height-1, termbox.ColorDefault, termbox.ColorDefault, line)
}

func (h *HandlerDefault) HandleKey(ev termbox.Event) {
	ui := h.ui
	vw, vh := ui.viewSize()

	maxYOffset := clamp(len(ui.filterMatches)-(vh-2), 0, len(ui.filterMatches)-1)
	endOfLine := ui.endOfLine()

	switch {
	case ev.Key == termbox.KeyCtrlL:
		termbox.Sync()
	case ev.Key == termbox.KeyCtrlA:
		ui.offsetX = 0
	case ev.Key == termbox.KeyCtrlE:
		ui.offsetX = -endOfLine + vw
	case ev.Key == termbox.KeyArrowRight:
		ui.offsetX = clamp(ui.offsetX-5, -endOfLine+vw, 0)
	case ev.Key == termbox.KeyArrowLeft:
		ui.offsetX = clamp(ui.offsetX+5, -endOfLine+vw, 0)
	case ev.Key == termbox.KeyArrowUp:
		ui.offsetY = clamp(ui.offsetY-1, 0, maxYOffset)
	case ev.Key == termbox.KeyArrowDown:
		ui.offsetY = clamp(ui.offsetY+1, 0, maxYOffset)
	case ev.Ch == '/', ev.Key == termbox.KeyCtrlR:
		ui.handler = &HandlerFilter{*h}
		ui.offsetY = 0
	case ev.Key == termbox.KeyEnter:
		jsonObj := make(map[string]interface{})

		row := ui.rows[ui.offsetY]
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

		ui.handler = NewPopup(h.ui, string(jsonStr))

	case ev.Key == termbox.KeySpace:
		ui.offsetY = clamp(ui.offsetY+vh, 0, maxYOffset)
	case ev.Ch == 'C':
		ui.handler = NewColumnSelect(h.ui)
		ui.offsetX = 0
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
	case ev.Ch == '?':
		ui.handler = NewPopup(h.ui, HELP_TEXT)
	}
}

type HandlerFilter struct {
	HandlerDefault
}

func (h *HandlerFilter) Repaint() {
	ui := h.ui
	_, height := termbox.Size()

	line := fmt.Sprintf("FILTER [%d matches]: %s", len(ui.filterMatches), ui.filterString)
	writeLine(0, height-1, termbox.ColorWhite|termbox.AttrBold, termbox.ColorDefault, line)
	termbox.SetCursor(len(line), height-1)

}

func (h *HandlerFilter) HandleKey(ev termbox.Event) {
	ui := h.ui

	// Ch == 0 implies this was a special key
	if ev.Ch == 0 && ev.Key != termbox.KeySpace {
		if ev.Key == termbox.KeyEsc || ev.Key == termbox.KeyCtrlG {
			ui.switchToDefault()

			ui.filterString = ""
			ui.filterRows(false)
		} else if ev.Key == termbox.KeyEnter {
			ui.switchToDefault()
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
			// FIXME: is this really the best way to do this in go?
			def := &HandlerDefault{h.ui}
			def.HandleKey(ev)
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

func (h *HandlerColumnSelect) HandleKey(ev termbox.Event) {
	ui := h.ui
	savedColumn := -1

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
		ui.columnOpts[h.column].collapsed = !ui.columnOpts[h.column].collapsed
	case ev.Ch == 'x':
		ui.columnOpts[h.column].expanded = !ui.columnOpts[h.column].expanded
		if ui.columnOpts[h.column].expanded {
			ui.columnOpts[h.column].collapsed = false
		}
	case ev.Ch == '.':
		ui.columnOpts[h.column].pinned = !ui.columnOpts[h.column].pinned

		if ui.columnOpts[h.column].pinned {
			ui.columnOpts[h.column].collapsed = false
		}

	case ev.Key == termbox.KeyCtrlG, ev.Key == termbox.KeyEsc:
		savedColumn = h.column
		h.selectColumn(-1)
		ui.switchToDefault()
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

		if colOpts.collapsed {
			cursorPosition += 1
		} else if !colOpts.expanded {
			cursorPosition += clamp(colOpts.width, 0, MAX_CELL_WIDTH)
		} else {
			cursorPosition += colOpts.width
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

func (h *HandlerPopup) Repaint() {
	width, height := termbox.Size()

	popupW := clamp(120, 40, width-5)
	popupH := clamp(len(h.content)+2, 10, height-5)

	x := width/2 - popupW/2
	y := height/2 - popupH/2

	fmtString := "%s%-" + strconv.Itoa(popupW) + "s%s"

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
			if i < len(h.content) {
				content = h.content[i]
			} else {
				content = " "
			}
		} else {
			border = borders[2]
			content = strings.Repeat("─", popupW)
		}

		line := fmt.Sprintf(fmtString, border[0], content, border[1])
		writeString(x, y+i, termbox.ColorWhite, termbox.ColorDefault, line)
	}

	line := "POPUP (^g quit)"
	writeLine(0, height-1, termbox.ColorWhite|termbox.AttrBold, termbox.ColorDefault, line)

}

func (h *HandlerPopup) HandleKey(ev termbox.Event) {

	if ev.Key == termbox.KeyEsc || ev.Key == termbox.KeyCtrlG || ev.Ch == 'q' {
		h.ui.handler = &HandlerDefault{h.ui}
	}
}
