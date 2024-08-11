package main

import (
	"database/sql"
	"domanscy.group/env"
	"fmt"
	"github.com/go-chi/chi"
	_ "github.com/mattn/go-sqlite3"
	"net/http"
)

type ServerConfig struct {
	ServerAddress string
	ServerPort    uint16

	EmailFromAddress string
	SmtpAddress      string
	SmtpPort         uint16

	DatabaseUrl string
}

func NewServer(cfg ServerConfig, db *sql.DB) http.Handler {
	r := chi.NewRouter()

	r.Post("/auth/login", HttpAuthLogin(&cfg, db))

	return r
}

func main() {
	cfg := ServerConfig{}
	env.ReadToCfg(&cfg, map[string]string{
		"ServerAddress":    "SERVER_ADDRESS",
		"ServerPort":       "SERVER_PORT",
		"SmtpAddress":      "SMTP_ADDRESS",
		"SmtpPort":         "SMTP_PORT",
		"EmailFromAddress": "EMAIL_FROM_ADDRESS",
		"DatabaseUrl":      "DATABASE_URL",
	})

	db, err := sql.Open("sqlite3", cfg.DatabaseUrl)
	if err != nil {
		panic(err)
	}

	handler := NewServer(cfg, db)

	err = http.ListenAndServe(fmt.Sprintf("%s:%d", cfg.ServerAddress, cfg.ServerPort), handler)
	if err != nil {
		panic(err)
	}
}
