package main

import (
	"context"
	"database/sql"
	"domanscy.group/env"
	"domanscy.group/parental-controls/server/database"
	"domanscy.group/parental-controls/server/users"
	"domanscy.group/simplecache"
	"fmt"
	"github.com/go-chi/chi"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"time"
)

type ServerConfig struct {
	AppUrl        string `env:"APP_URL"`
	ServerAddress string `env:"SERVER_ADDRESS"`
	ServerPort    uint16 `env:"SERVER_PORT"`

	EmailFromAddress string `env:"EMAIL_FROM_ADDRESS"`
	SmtpAddress      string `env:"SMTP_ADDRESS"`
	SmtpPort         uint16 `env:"SMTP_PORT"`

	DatabaseUrl string `env:"DATABASE_URL"`
}

func NewServer(cfg ServerConfig, regkeysStore *simplecache.Store, db *sql.DB) http.Handler {
	r := chi.NewRouter()

	r.Post("/login", HttpAuthLogin(&cfg, regkeysStore, db))
	r.Post("/register", HttpAuthStartRegistrationProcess(&cfg, regkeysStore, db))

	return r
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	regkeysStore := simplecache.InitializeStore(ctx, time.Minute*15)

	cfg := ServerConfig{}
	env.ReadToCfg(&cfg)

	// remove trailing slash
	if cfg.AppUrl[len(cfg.AppUrl)-1] == '/' {
		cfg.AppUrl = cfg.AppUrl[:len(cfg.AppUrl)-1]
	}

	db, err := sql.Open("sqlite3", cfg.DatabaseUrl)
	if err != nil {
		panic(err)
	}

	err = database.Migrate(db, map[string]string{
		"0001_users": users.MigrationFile,
	})
	if err != nil {
		panic(err)
	}

	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Println(err)
		}
	}(db)

	handler := NewServer(cfg, regkeysStore, db)

	err = http.ListenAndServe(fmt.Sprintf("%s:%d", cfg.ServerAddress, cfg.ServerPort), handler)
	if err != nil {
		panic(err)
	}
}
