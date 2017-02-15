package main

import "strings"
import "github.com/nsf/termbox-go"

func writeString(x, y int, fg, bg termbox.Attribute, msg string) {
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
		x += 1
	}
}

func UiLoop(data TabularData) {
	if err := termbox.Init(); err != nil {
		panic(err)
	}

	defer termbox.Close()
	termbox.SetInputMode(termbox.InputEsc)

	const coldef = termbox.ColorDefault
	writeString(0, 0, coldef, coldef, strings.Join(data.Columns, " ┋ "))

	//writeString(0, 0)

	for i, row := range data.Rows {
		writeString(0, 1+i, coldef, coldef, strings.Join(row, " │ "))
	}

	termbox.Flush()

	switch ev := termbox.PollEvent(); ev.Type {
	case termbox.EventKey:
		break
	}
}
