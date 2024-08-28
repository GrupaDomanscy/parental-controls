package components

import (
	"github.com/chasefleming/elem-go"
	"github.com/chasefleming/elem-go/attrs"
)

type GetUrlCallback func(name string, args map[string]interface{}) string

func LoginPage(getUrl GetUrlCallback) *elem.Element {
	return Template(
		elem.Div(attrs.Props{
			attrs.Class: "flex flex-col min-h-screen",
		},
			Navbar(),
			elem.Div(attrs.Props{
				attrs.Class: "w-full flex-grow flex justify-center",
			},
				elem.Div(attrs.Props{
					attrs.Class: "w-full max-w-lg rounded p-4 m-2 gap-4 flex flex-col",
				},
					elem.H2(attrs.Props{
						attrs.Class: "text-2xl font-semibold text-center",
					}, elem.Text("Logowanie")),

					elem.Div(attrs.Props{
						attrs.Class: "flex flex-col gap-1",
					},
						elem.Label(attrs.Props{
							attrs.Class: "w-full text-neutral-200",
							attrs.For:   "email",
						}, elem.Text("Adres email")),

						elem.Input(attrs.Props{
							attrs.Class: "w-full p-2 bg-zinc-900/80 rounded focus:outline focus:outline-blue-500",
							attrs.Name:  "email",
							attrs.Type:  "email",
						}),
					),

					elem.Div(attrs.Props{
						attrs.Class: "flex flex-col gap-1 justify-center items-center",
					},
						elem.Button(attrs.Props{
							attrs.Class: "px-4 py-2 bg-blue-700/50 hover:bg-blue-600 focus:bg-blue-600 focus:outline focus:outline-blue-500 rounded w-full",
						}, elem.Text("Zaloguj")),
						elem.A(attrs.Props{
							attrs.Class: "px-4 py-2 text-blue-300",
							attrs.Href:  getUrl("register", nil),
						}, elem.Text("Nie masz konta? Kliknij tutaj.")),
					),
				),
			),
		),
	)
}
