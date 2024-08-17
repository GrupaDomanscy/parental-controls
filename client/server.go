package client

import (
	"context"
	"embed"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

//go:embed assets/*
var assetsFs embed.FS

type ServerOpts struct {
	smtpAddress string
	smtpPort    uint16
}

func NewServer(opts ServerOpts) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	assetsFsStripped, err := fs.Sub(assetsFs, "assets")
	if err != nil {
		panic("Could not strip 'assets' directory name from assets filesystem.")
	}

	r.Get("/assets/{asset}", func(w http.ResponseWriter, r *http.Request) {
		asset := chi.URLParam(r, "asset")

		http.ServeFileFS(w, r, assetsFsStripped, asset)
	})
	r.Get("/", RenderLoginPageHttpHandler(opts))
	r.Post("/", LoginHttpHandler(opts))

	return r
}

func StartServer(port uint16, ctx context.Context) {
	http.ListenAndServe(":8080", NewServer(ServerOpts{
		smtpAddress: "127.0.0.1",
		smtpPort:    1025,
	}))
}
