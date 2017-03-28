// ui mode specific handlers

package vxsv

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"unicode"

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
	ui.writeModeLine(":", []string{})
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
	case unicode.ToLower(ev.Ch) == 'c':
		ui.pushHandler(NewColumnSelect(h.ui))
		ui.offsetX = 0
	case unicode.ToLower(ev.Ch) == 'r':
		ui.pushHandler(&HandlerRowSelect{*h, h.ui.offsetY})
	case ev.Ch == 'G':
		ui.offsetY = maxYOffset
	case ev.Ch == 'g':
		ui.offsetY = 0
	case ev.Ch == 'Z':
		ui.zebraStripe = !ui.zebraStripe
	case ev.Ch == 'X':
		var displayMode ColumnDisplay = ColumnExpanded

		if ui.allExpanded {
			displayMode = ColumnDefault
		}

		for i := range ui.columns {
			ui.columns[i].Display = displayMode
		}

		ui.allExpanded = !ui.allExpanded
	case ev.Ch == '?':
		ui.pushHandler(NewPopup(h.ui, HelpText))
	}
}

type HandlerFilter struct {
	HandlerDefault
	filter string
}

func (h *HandlerFilter) Repaint() {
	ui := h.ui
	_, height := termbox.Size()

	ui.writeModeLine("Filter", []string{h.filter})
	termbox.SetCursor(len("filter")+1+len(h.filter), height-1)
}

func handlePromptKey(ev termbox.Event, str *string) (consumed bool) {
	if ev.Key == termbox.KeyDelete || ev.Key == termbox.KeyBackspace || ev.Key == termbox.KeyBackspace2 {
		if sz := len(*str); sz > 0 {
			*str = (*str)[:sz-1]
		}
	} else if ev.Key == termbox.KeyCtrlW || ev.Key == termbox.KeyCtrlU {
		*str = ""
	} else if ev.Key == termbox.KeySpace {
		*str += " "
	} else if ev.Ch != 0 {
		*str += string(ev.Ch)
	} else {
		// Unknown key press
		return false
	}

	return true
}

func (h *HandlerFilter) HandleKey(ev termbox.Event) {
	ui := h.ui

	if handlePromptKey(ev, &h.filter) {
		return
	} else if ev.Key == termbox.KeyEsc || ev.Key == termbox.KeyCtrlG {
		ui.popHandler()

		ui.filter = EmptyFilter{}
		ui.filterRows()
	} else if ev.Key == termbox.KeyEnter {
		if h.filter == "" {
			ui.filter = EmptyFilter{}
		} else if filter, err := ui.parseFilter(h.filter); err == nil {
			ui.filter = filter
		} else {
			ui.pushErrorPopup("There was an error in your filter: "+h.filter, err)
			return
		}
		ui.filterRows()
		ui.popHandler()
	} else {
		// Fallback to default handling for arrows etc
		// FIXME: is this really the best way to do this in go?
		def := &HandlerDefault{h.ui}
		def.HandleKey(ev)
	}
}

type HandlerShell struct {
	HandlerDefault
	colIdx  int
	command string
}

func (h *HandlerShell) applyCommand() {
	cmd := exec.Command("sh", "-c", h.command)

	in, err := cmd.StdinPipe()
	if err != nil {
		// FIXME: more context
		panic(err)
	}

	out, err := cmd.StdoutPipe()
	if err != nil {
		// FIXME: more context
		panic(err)
	}

	errOut, err := cmd.StderrPipe()
	if err != nil {
		// FIXME: more context
		panic(err)
	}

	go func() {
		defer in.Close()

		for i := range h.ui.rows {
			row := h.ui.getRow(i)
			io.WriteString(in, row[h.colIdx]+"\n")
		}
	}()

	if err := cmd.Start(); err != nil {
		panic(err)
	}

	modifiedColumn := make([]string, len(h.ui.rows))

	scanner := bufio.NewScanner(out)
	for i := 0; i < len(h.ui.rows); i++ {
		if !scanner.Scan() {
			output, _ := ioutil.ReadAll(errOut)
			h.ui.pushErrorPopup("Process exited too early!", fmt.Errorf("%s", output))
			break
		}

		if err := scanner.Err(); err != nil {
			h.ui.pushErrorPopup("There was an error running your command:", err)
			break
		}

		modifiedColumn[i] = scanner.Text()
	}

	h.ui.columns[h.colIdx].Modified = true
	h.ui.columns[h.colIdx].ModifiedValues = modifiedColumn
	h.ui.columns[h.colIdx].ModifiedCommand = h.command
}

func (h *HandlerShell) HandleKey(ev termbox.Event) {
	if handlePromptKey(ev, &h.command) {
		return
	} else if ev.Key == termbox.KeyEsc || ev.Key == termbox.KeyCtrlG {
		h.ui.columns[h.colIdx].Modified = false
		h.ui.popHandler()
	} else if ev.Key == termbox.KeyEnter {
		trimmed := strings.TrimSpace(h.command)
		h.ui.popHandler()

		if len(trimmed) > 0 {
			h.applyCommand()
		} else {
			h.ui.columns[h.colIdx].Modified = false
		}
	}

	h.ui.recomputeColumnWidth(h.colIdx)
}

func (h *HandlerShell) Repaint() {
	_, height := termbox.Size()

	h.ui.writeModeLine("Run shell", []string{h.command})
	termbox.SetCursor(len("run shell")+1+len(h.command), height-1)
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

		rowIdx := ui.filterMatches[h.rowIdx]
		row := ui.getRow(rowIdx)

		for i, col := range ui.columns {
			str := row[i]

			if v, err := strconv.ParseInt(str, 10, 64); err == nil {
				jsonObj[col.Name] = v
			} else if v, err := strconv.ParseFloat(str, 64); err == nil {
				jsonObj[col.Name] = v
			} else if v, err := strconv.ParseBool(str); err == nil {
				jsonObj[col.Name] = v
			} else {
				jsonObj[col.Name] = str
			}
		}

		if jsonStr, err := json.MarshalIndent(jsonObj, "", "  "); err == nil {
			ui.pushHandler(NewPopup(ui, string(jsonStr)))
		} else {
			ui.pushErrorPopup("Failed to dump row as json (this is a bug)", err)
		}
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

	termbox.SetCell(0, 1+h.rowIdx-ui.offsetY, RowIndicator, termbox.ColorRed|termbox.AttrBold, termbox.ColorWhite)
	ui.writeModeLine("Row Select", []string{strconv.Itoa(h.rowIdx)})
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
	h.ui.columns[h.column].Highlight = false
	h.column = idx

	if h.column >= 0 {
		h.ui.columns[h.column].Highlight = true
	}
}

func (h *HandlerColumnSelect) rowSorter(i, j int) bool {
	ui := h.ui

	row1 := ui.getRow(ui.filterMatches[i])
	row2 := ui.getRow(ui.filterMatches[j])

	v1, err1 := strconv.ParseFloat(row1[h.column], 32)
	v2, err2 := strconv.ParseFloat(row2[h.column], 32)

	if err1 == nil && err2 == nil {
		return v1 < v2
	}

	return row1[h.column] < row2[h.column]
}

func (h *HandlerColumnSelect) Repaint() {
	ui := h.ui

	col := fmt.Sprintf("[%s]", ui.columns[h.column].Name)
	ui.writeModeLine("Column Select", []string{col})
}

func (h *HandlerColumnSelect) HandleKey(ev termbox.Event) {
	ui := h.ui
	col := &ui.columns[h.column]

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
	case unicode.ToLower(ev.Ch) == 'c':
		h.selectColumn(0)
	case ev.Ch == 'w':
		col.toggleDisplay(ColumnCollapsed)
	case ev.Ch == 'x':
		col.toggleDisplay(ColumnExpanded)
		ui.recomputeColumnWidth(h.column)
	case ev.Ch == 'a':
		col.toggleDisplay(ColumnAligned)
		ui.recomputeColumnWidth(h.column)
	case ev.Ch == '.':
		col.Pinned = !col.Pinned

		if col.Pinned {
			col.Display = ColumnDefault
		}
	case ev.Ch == '|':
		commandStr := ui.columns[h.column].ModifiedCommand
		h.ui.pushHandler(&HandlerShell{HandlerDefault{ui}, h.column, commandStr})
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

		colName := ui.columns[h.column].Name

		data := make(stats.Float64Data, 0, len(ui.filterMatches)+1)
		for _, rowIdx := range ui.filterMatches {
			row := ui.getRow(rowIdx)
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
  [ %s ]
  %s
  rows visible: %d (of %d)
  numeric rows: %d

  min: %15.4f      mean:   %15.4f
  max: %15.4f      median: %15.4f
  sum: %15.4f      mode:   %15.4f

  var: %15.4f      std:    %15.4f

  p90: %15.4f      p25:    %15.4f
  p95: %15.4f      p50:    %15.4f
  p99: %15.4f      p75:    %15.4f`,
			colName, strings.Repeat("-", 4+len(colName)),
			len(ui.filterMatches), len(ui.rows), len(data),
			min, mean, max, median, sum, mode, variance, stdev,
			p90, quartiles.Q1, p95, quartiles.Q2, p99, quartiles.Q3)

		ui.pushHandler(NewPopup(ui, text))

		break
	error:
		ui.pushErrorPopup("Summary stats failed! (probably a bug)", err)
	case ev.Key == termbox.KeyCtrlG, ev.Key == termbox.KeyEsc:
		h.selectColumn(-1)
		ui.popHandler()
		return
	default:
		// FIXME: ditto, is this the best way to do this?
		def := &HandlerDefault{h.ui}
		def.HandleKey(ev)
	}

	// find if we've gone off screen and readjust
	// TODO: this bit is buggy when scrolling right
	columnOffset, colWidth := ui.columnOffset(h.column)
	width, _ := termbox.Size()
	viewWidth, _ := ui.viewSize()

	if ui.offsetX+width < columnOffset || columnOffset-colWidth < ui.offsetX {
		ui.offsetX = columnOffset - colWidth
	}

	lastColumnOffset, _ := ui.columnOffset(len(ui.columns) - 1)
	ui.offsetX = clamp(ui.offsetX, 0, lastColumnOffset-viewWidth)
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

	for i := -1; i <= popupH; i++ {
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

			// Horizontal scrolling
			if h.offsetX > len(content) {
				content = ""
			} else {
				content = content[h.offsetX:]
			}

		} else {
			border = borders[2]
			content = strings.Repeat("─", popupW)
		}

		line := fmt.Sprintf("%s%-*s%s", border[0], popupW, content, border[1])
		writeString(x, y+i, termbox.ColorDefault, termbox.ColorDefault, line)
	}

	h.ui.writeModeLine("Modal", []string{})
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
