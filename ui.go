package main

import "strings"
import "github.com/nsf/termbox-go"

const MAX_CELL_WIDTH = 20
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

func writeString(x, y int, fg, bg termbox.Attribute, msg string) {
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
		x += 1
	}
}

func (ui *UI) writeColumns(x, y int) {
	var fg, bg termbox.Attribute

	for i, col := range ui.data.Columns {
		if i == ui.colIdx && ui.mode == ModeColumnSelect {
			fg = HILITE_FG
			bg = HILITE_BG
		} else {
			fg = termbox.ColorWhite
			bg = termbox.ColorBlack | termbox.AttrBold
		}

		if col.Collapsed {
			writeString(x, y, fg, bg, "…")
			x += 1
		} else {
			writeString(x, y, fg, bg, col.Name)

			if col.Width > MAX_CELL_WIDTH {
				x += MAX_CELL_WIDTH
			} else {
				x += col.Width
			}
		}

		if i < len(ui.data.Columns)-1 {
			writeString(x, y, termbox.ColorWhite, termbox.ColorDefault, " │ ")
			x += 3
		}
	}
}

func (ui *UI) writeRow(x, y int, row []string) {
	const def = termbox.ColorDefault
	var fg, bg termbox.Attribute

	for i, col := range ui.data.Columns {
		if i == ui.colIdx && ui.mode == ModeColumnSelect {
			fg = HILITE_FG
			bg = HILITE_BG
		} else {
			fg = def
			bg = def
		}

		if col.Collapsed {
			writeString(x, y, fg, bg, "…")
			x += 1
		} else {
			writeString(x, y, fg, bg, row[i])
			if col.Width > MAX_CELL_WIDTH {
				writeString(x+MAX_CELL_WIDTH-1, y, fg, bg, "…")
				x += MAX_CELL_WIDTH
			} else {
				x += col.Width
			}
		}

		if i < len(ui.data.Columns)-1 {
			writeString(x, y, termbox.ColorRed, def, " │ ")
			x += 3
		}
	}
}

type UI struct {
	data             TabularData
	mode             inputMode
	rowIdx, colIdx   int // Selection control
	offsetX, offsetY int // Pan control
	filterString     string
}

func NewUi(data TabularData) UI {
	return UI{
		data:    data,
		offsetX: 0,
		offsetY: 0,
		mode:    ModeDefault,
		colIdx:  -1,
	}
}

func (ui *UI) Init() error {
	if err := termbox.Init(); err != nil {
		return err
	}

	termbox.SetInputMode(termbox.InputEsc | termbox.InputMouse)

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
func (ui *UI) filterRows(num int) []int {
	rows := make([]int, 0, num)

	if ui.mode != ModeFilter {
		for i := 0; i < num; i += 1 {
			if i+ui.offsetY >= len(ui.data.Rows) {
				break
			}
			rows = append(rows, i+ui.offsetY)
		}
	} else {
		for i := 0; i < num; i += 1 {
			if i+ui.offsetY >= len(ui.data.Rows) {
				break
			}

			for _, col := range ui.data.Rows[i+ui.offsetY] {
				if strings.Contains(col, ui.filterString) {
					rows = append(rows, i+ui.offsetY)
					break
				}
			}
		}
	}

	return rows
}

func (ui *UI) repaint() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	_, height := termbox.Size()

	const coldef = termbox.ColorDefault

	ui.writeColumns(ui.offsetX+0, 0)

	rowIdx := ui.filterRows(height - 2)

	for i := 0; i < height-2; i += 1 {
		if i < len(rowIdx) {
			ui.writeRow(ui.offsetX+0, i+1, ui.data.Rows[rowIdx[i]])
		} else {
			writeString(0, i+1, termbox.ColorWhite|termbox.AttrBold, termbox.ColorBlack, "~")
		}
	}

	switch ui.mode {
	case ModeFilter:
		line := "FILTER (^g quit): " + ui.filterString
		writeString(0, height-1, termbox.ColorWhite|termbox.AttrBold, termbox.ColorDefault, line)
	case ModeColumnSelect:
		line := "COLUMN SELECT (^g quit) [" + ui.data.Columns[ui.colIdx].Name + "]"
		writeString(0, height-1, termbox.ColorWhite|termbox.AttrBold, termbox.ColorDefault, line)
	default:
		writeString(0, height-1, termbox.ColorDefault, termbox.ColorDefault, ":")
	}

	termbox.Flush()
}

func (ui *UI) handleKeyFilter(ev termbox.Event) {
	// Ch == 0 implies this was a special key
	if ev.Ch == 0 && ev.Key != termbox.KeySpace {
		if ev.Key == termbox.KeyEsc || ev.Key == termbox.KeyCtrlG {
			ui.mode = ModeDefault
		} else if ev.Key == termbox.KeyDelete || ev.Key == termbox.KeyBackspace ||
			ev.Key == termbox.KeyBackspace2 {
			if sz := len(ui.filterString); sz > 0 {
				ui.filterString = ui.filterString[:sz-1]
			}
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
}

func (ui *UI) handleKeyColumnSelect(ev termbox.Event) {
	switch {
	case ev.Key == termbox.KeyArrowRight:
		ui.colIdx = clamp(ui.colIdx+1, 0, len(ui.data.Columns)-1)
	case ev.Key == termbox.KeyArrowLeft:
		ui.colIdx = clamp(ui.colIdx-1, 0, len(ui.data.Columns)-1)
	case ev.Ch == 'w':
		ui.data.Columns[ui.colIdx].Collapsed = !ui.data.Columns[ui.colIdx].Collapsed
	case ev.Key == termbox.KeyCtrlG, ev.Key == termbox.KeyEsc:
		ui.mode = ModeDefault
	default:
		ui.handleKeyDefault(ev)
	}

	// find if we've gone off screen and readjust
	cursorPosition := 0
	for i, col := range ui.data.Columns {
		if i == ui.colIdx {
			break
		}
		//cursorPosition += 3
		if !col.Collapsed {
			cursorPosition += col.Width
		}
	}

	width, _ := termbox.Size()
	if cursorPosition > width-ui.offsetX || cursorPosition < -ui.offsetX {
		ui.offsetX = -cursorPosition
	}
}

func (ui *UI) handleKeyDefault(ev termbox.Event) {
	switch {
	case ev.Key == termbox.KeyArrowRight:
		ui.offsetX = clamp(ui.offsetX-5, -ui.data.Width, 0)
	case ev.Key == termbox.KeyArrowLeft:
		ui.offsetX = clamp(ui.offsetX+5, -ui.data.Width, 0)
	case ev.Key == termbox.KeyArrowUp:
		ui.offsetY = clamp(ui.offsetY-1, 0, len(ui.data.Rows))
	case ev.Key == termbox.KeyArrowDown:
		ui.offsetY = clamp(ui.offsetY+1, 0, len(ui.data.Rows))
	case ev.Ch == '/':
		ui.mode = ModeFilter
		ui.filterString = ""
		ui.offsetY = 0
	case ev.Ch == 'C':
		ui.mode = ModeColumnSelect
		ui.offsetX = 0
		ui.colIdx = 0
	case ev.Ch == 'G':
		_, height := termbox.Size()
		ui.offsetY = len(ui.data.Rows) - (height - 3)
	case ev.Ch == 'g':
		ui.offsetY = 0
	case ui.mode == ModeDefault && ev.Ch == 'q':
		panic("TODO: real exit")
	}
}
