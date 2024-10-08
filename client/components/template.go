package components

import (
	"embed"
	"fmt"

	"github.com/chasefleming/elem-go"
	"github.com/chasefleming/elem-go/attrs"
)

//go:generate cp -r ../assets local-assets-dir
//go:embed local-assets-dir
var assetsFS embed.FS

var (
	appCssFileTimestamp     int64
	tailwindJsFileTimestamp int64
	// timestampsMutex         sync.Mutex
)

func init() {
	file, err := assetsFS.Open("local-assets-dir/app.css")
	if err != nil {
		panic(err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		panic(err)
	}

	appCssFileTimestamp = fileInfo.ModTime().UnixMilli()

	file, err = assetsFS.Open("local-assets-dir/tailwindcss.js")
	if err != nil {
		panic(err)
	}

	fileInfo, err = file.Stat()
	if err != nil {
		panic(err)
	}

	tailwindJsFileTimestamp = fileInfo.ModTime().UnixMilli()
}

func Template(elements ...elem.Node) *elem.Element {
	// timestampsMutex.Lock()
	// defer timestampsMutex.Unlock()

	return elem.Html(attrs.Props{
		attrs.Lang: "pl",
	},
		elem.Head(nil,
			elem.Title(nil, elem.Text("Parental controls")),

			elem.Meta(attrs.Props{
				attrs.Name:    "unicode",
				attrs.Charset: "utf-8",
			}),

			elem.Meta(attrs.Props{
				attrs.Name:    "viewport",
				attrs.Content: "width=device-width, initial-scale=1",
			}),

			elem.Script(attrs.Props{
				attrs.Src: fmt.Sprintf("/assets/tailwindcss.js?%d", tailwindJsFileTimestamp),
			}),

			elem.Link(attrs.Props{
				attrs.Href: fmt.Sprintf("/assets/app.css?%d", appCssFileTimestamp),
				attrs.Rel:  "stylesheet",
			}),
		),
		elem.Body(attrs.Props{
			attrs.Class: "bg-zinc-800 min-h-screen w-full text-neutral-100",
		}, elements...),
	)
}
