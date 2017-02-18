package main

import "strings"
import "github.com/nsf/termbox-go"

const MAX_CELL_WIDTH = 20

type inputMode int

const (
	ModeDefault = iota
	ModeFilter
	ModeColumnSelect
	ModeRowSelect
)

func writeString(x, y int, fg, bg termbox.Attribute, msg string) {
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
		x += 1
	}
}

func writeColumns(x, y int, fg, bg termbox.Attribute, cols []Column) {
	for i, col := range cols {
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

		if i < len(cols)-1 {
			writeString(x, y, fg, bg, " │ ")
			x += 3
		}
	}
}

func (ui UI) writeRow(x, y int, row []string) {
	const def = termbox.ColorDefault

	for i, col := range ui.data.Columns {
		if col.Collapsed {
			writeString(x, y, termbox.ColorWhite, def, "…")
			x += 1
		} else {
			writeString(x, y, def, def, row[i])
			if col.Width > MAX_CELL_WIDTH {
				writeString(x+MAX_CELL_WIDTH-1, y, def, def, "…")
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

	writeColumns(ui.offsetX+0, 0, termbox.ColorWhite, termbox.ColorBlack|termbox.AttrBold, ui.data.Columns)

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

func (ui *UI) handleKeyDefault(ev termbox.Event) {
	switch {
	case ev.Key == termbox.KeyArrowRight:
		ui.offsetX -= 5
	case ev.Key == termbox.KeyArrowLeft:
		if ui.offsetX < 0 {
			ui.offsetX += 5
		}
	case ev.Key == termbox.KeyArrowUp:
		if ui.offsetY > 0 {
			ui.offsetY -= 1
		}
	case ev.Key == termbox.KeyArrowDown:
		ui.offsetY += 1
	case ev.Ch == '/':
		ui.mode = ModeFilter
		ui.filterString = ""
	case ev.Ch == 'w':
		ui.data.Columns[0].Collapsed = !ui.data.Columns[0].Collapsed
	case ui.mode == ModeDefault && ev.Ch == 'q':
		panic("TODO: real exit")
	}
}
