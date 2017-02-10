package main

import "github.com/nsf/termbox-go"

func writeString(x, y int, fg, bg termbox.Attribute, msg string) {
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
		x += 1
	}
}

func UiLoop() {
	if err := termbox.Init(); err != nil {
		panic(err)
	}

	defer termbox.Close()
	termbox.SetInputMode(termbox.InputEsc)

	const coldef = termbox.ColorDefault
	writeString(10, 10, coldef, coldef, "test")

	termbox.Flush()

	switch ev := termbox.PollEvent(); ev.Type {
	case termbox.EventKey:
		break
	}
}
