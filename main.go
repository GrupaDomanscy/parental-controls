package main

import (
	"domanscy.group/gui"
	. "domanscy.group/gui/components"
)

func main() {
	text := NewTextComponent("Hello world! How are you today? Hello world! How are you today? Hello world! How are you today?", "Roboto", 32, 0, gui.WhiteColor)
	text.SetWrapText(true)

	gui.BuildApp().
		WithTitle("Hello world").
		WithInitialSize(800, 600).
		WithFont("Roboto", "assets/Roboto-Regular.ttf").
		WithRootElement(text).
		Run()
}
