package main

import "github.com/nsf/termbox-go"

const MAX_CELL_WIDTH = 20

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
			writeString(x, y, def, termbox.ColorWhite, "…")
			x += 1
		} else {
			writeString(x, y, def, def, row[i])
			if col.Width > MAX_CELL_WIDTH {
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
	offsetX, offsetY int
}

func NewUi(data TabularData) UI {
	return UI{
		data:    data,
		offsetX: 0,
		offsetY: 0,
	}
}

func (ui UI) Init() error {
	if err := termbox.Init(); err != nil {
		return err
	}

	termbox.SetInputMode(termbox.InputEsc | termbox.InputMouse)

	return nil
}

func (ui UI) Loop() {
	defer termbox.Close()

	ui.repaint()

eventloop:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			if ev.Key == termbox.KeyEsc || ev.Ch == 'q' || ev.Key == termbox.KeyCtrlC {
				break eventloop
			} else if ev.Key == termbox.KeyArrowRight {
				ui.offsetX -= 5
				ui.repaint()
			} else if ev.Key == termbox.KeyArrowLeft {
				if ui.offsetX < 0 {
					ui.offsetX += 5
					ui.repaint()
				}
			} else if ev.Ch == 'w' {
				ui.data.Columns[0].Collapsed = !ui.data.Columns[0].Collapsed
				ui.repaint()
			}

		}
	}
}

func (ui UI) repaint() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	_, height := termbox.Size()

	const coldef = termbox.ColorDefault

	writeColumns(ui.offsetX+0, 0, termbox.ColorWhite, termbox.ColorBlack|termbox.AttrBold, ui.data.Columns)

	for i := 0; i < height-1; i += 1 {
		if i < len(ui.data.Rows) {
			ui.writeRow(ui.offsetX+0, i+1, ui.data.Rows[i])
		} else {
			writeString(0, i+1, termbox.ColorWhite, termbox.ColorBlack|termbox.AttrBold, "~")
		}
	}

	termbox.Flush()
}
