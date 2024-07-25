package main

import (
	"domanscy.group/gui"
	. "domanscy.group/gui/components"
)

const LoremIpsum = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Phasellus id sapien eu sem dignissim pharetra. Etiam laoreet massa sit amet ornare mollis. Ut cursus sit amet magna eu tempor. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia curae; Pellentesque eleifend eros quis sapien ultricies, in cursus purus auctor. Quisque in sem vel dui blandit condimentum eu ac tortor. Sed a turpis a ligula auctor sagittis. Nulla tincidunt libero eros, eget suscipit enim facilisis vitae. Duis quis est eu tellus eleifend condimentum. Ut id nunc nec velit pellentesque rhoncus. Proin urna odio, tincidunt ac sapien et, egestas posuere elit. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nam blandit tincidunt dolor eu tincidunt. Proin dignissim arcu justo, eget ornare est iaculis eget. Nunc quis volutpat quam, vitae ornare metus."

func main() {
	text := NewTextComponent(LoremIpsum, "Roboto", 32, 0, gui.WhiteColor)

	gui.BuildApp().
		WithTitle("Hello world").
		WithInitialSize(800, 600).
		WithFont("Roboto", "assets/Roboto-Regular.ttf").
		WithRootElement(text).
		Run()
}
