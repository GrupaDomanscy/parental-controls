package client

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi"
)

//go:embed assets/*
var assetsFs embed.FS

type ServerOpts struct {
	smtpAddress string
	smtpPort    uint16
	baseUrl     string
}

func NewServer(opts ServerOpts) http.Handler {
	router := chi.NewRouter()

	assetsFsStripped, err := fs.Sub(assetsFs, "assets")
	if err != nil {
		panic("Could not strip 'assets' directory name from assets filesystem.")
	}

	router.Get("/assets/{asset}", func(w http.ResponseWriter, r *http.Request) {
		asset := chi.URLParam(r, "asset")
		http.ServeFileFS(w, r, assetsFsStripped, asset)
	})
	router.Get("/login", RenderLoginPageHttpHandler(opts))
	router.Post("/login", LoginHttpHandler(opts))

	return router
}

func StartServer(port uint16) {
	http.ListenAndServe(fmt.Sprintf(":%d", port), NewServer(ServerOpts{
		smtpAddress: "127.0.0.1",
		smtpPort:    8080,
		baseUrl:     "http://localhost:8080",
	}))
}
