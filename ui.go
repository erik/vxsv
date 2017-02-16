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
		writeString(x, y, fg, bg, col.Name)

		if col.Width > MAX_CELL_WIDTH {
			x += MAX_CELL_WIDTH
		} else {
			x += col.Width
		}

		if i < len(cols)-1 {
			writeString(x, y, fg, bg, " ┋ ")
			x += 3
		}
	}
}

func writeRow(x, y int, fg, bg termbox.Attribute, cols []Column, row []string) {
	for i, col := range cols {
		writeString(x, y, fg, bg, row[i])

		if col.Width > MAX_CELL_WIDTH {
			x += MAX_CELL_WIDTH
		} else {
			x += col.Width
		}

		if i < len(cols)-1 {
			writeString(x, y, fg, bg, " ┋ ")
			x += 3
		}
	}
}

func UiLoop(data TabularData) {
	if err := termbox.Init(); err != nil {
		panic(err)
	}

	defer termbox.Close()
	termbox.SetInputMode(termbox.InputEsc)

	const coldef = termbox.ColorDefault
	writeColumns(0, 0, coldef, coldef, data.Columns)

	//writeString(0, 0)

	for i, row := range data.Rows {
		writeRow(0, i+1, coldef, coldef, data.Columns, row)
	}

	termbox.Flush()

	switch ev := termbox.PollEvent(); ev.Type {
	case termbox.EventKey:
		break
	}
}
