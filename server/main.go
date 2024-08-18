package main

import (
	"database/sql"
	"domanscy.group/env"
	"fmt"
	"github.com/go-chi/chi"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
)

type ServerConfig struct {
	ServerAddress string `env:"SERVER_ADDRESS"`
	ServerPort    uint16 `env:"SERVER_PORT"`

	EmailFromAddress string `env:"EMAIL_FROM_ADDRESS"`
	SmtpAddress      string `env:"SMTP_ADDRESS"`
	SmtpPort         uint16 `env:"SMTP_PORT"`

	DatabaseUrl string `env:"DATABASE_URL"`
}

func NewServer(cfg ServerConfig, db *sql.DB) http.Handler {
	r := chi.NewRouter()

	r.Post("/login", HttpAuthLogin(&cfg, db))
	r.Post("/register", HttpAuthStartRegistrationProcess(&cfg, db))

	return r
}

func main() {
	cfg := ServerConfig{}
	env.ReadToCfg(&cfg)

	db, err := sql.Open("sqlite3", cfg.DatabaseUrl)
	if err != nil {
		panic(err)
	}

	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Println(err)
		}
	}(db)

	handler := NewServer(cfg, db)

	err = http.ListenAndServe(fmt.Sprintf("%s:%d", cfg.ServerAddress, cfg.ServerPort), handler)
	if err != nil {
		panic(err)
	}
}
