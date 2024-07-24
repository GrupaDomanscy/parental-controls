package main

import (
	"domanscy.group/gui"
	. "domanscy.group/gui/components"
)

func main() {
	layout := NewLayoutComponent(DirectionRow, AlignStart, AlignStart)

	text := NewTextComponent("Hello world! How are you today?", "Roboto", 32, 0, gui.WhiteColor)
	text2 := NewTextComponent("Hello world! How are you today?", "Roboto", 32, 0, gui.WhiteColor)
	text3 := NewTextComponent("Hello world! How are you today?", "Roboto", 32, 0, gui.WhiteColor)

	layout.AddChild(text)
	layout.AddChild(text2)
	layout.AddChild(text3)

	gui.BuildApp().
		WithTitle("Hello world").
		WithInitialSize(800, 600).
		WithFont("Roboto", "assets/Roboto-Regular.ttf").
		WithRootElement(layout).
		Run()
}
