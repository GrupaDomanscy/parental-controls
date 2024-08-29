package components

import (
	"github.com/chasefleming/elem-go"
	"github.com/chasefleming/elem-go/attrs"
)

func Navbar() *elem.Element {
	return elem.Div(attrs.Props{attrs.Class: "flex flex-row justify-between items-center p-4 bg-zinc-800 shadow-xl text-neutral-100"},
		elem.H1(attrs.Props{
			attrs.Class: "text-2xl",
		}, elem.Text("Parental controls")),
		elem.Button(attrs.Props{
			attrs.Type:  "button",
			attrs.Class: "",
		}, MenuSvg()),
	)
}
