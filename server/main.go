package main

import (
	"database/sql"
	"domanscy.group/env"
	"domanscy.group/parental-controls/server/database"
	"domanscy.group/parental-controls/server/users"
	"domanscy.group/rckstrvcache"
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

func NewServer(cfg ServerConfig, regkeysStore *rckstrvcache.Store, oneTimeAccessTokenStore *rckstrvcache.Store, db *sql.DB) http.Handler {
	r := chi.NewRouter()

	r.Post("/login", HttpAuthLogin(&cfg, regkeysStore, oneTimeAccessTokenStore, db))
	r.Post("/register", HttpAuthStartRegistrationProcess(&cfg, regkeysStore, oneTimeAccessTokenStore, db))
	r.Get("/finish_registration/{regkey}", HttpAuthFinishRegistrationProcess(&cfg, regkeysStore, oneTimeAccessTokenStore, db))

	return r
}

func startServer(cfg ServerConfig, regkeysStore *rckstrvcache.Store, otatStore *rckstrvcache.Store, db *sql.DB, errCh chan<- error) {
	handler := NewServer(cfg, regkeysStore, otatStore, db)

	err := http.ListenAndServe(fmt.Sprintf("%s:%d", cfg.ServerAddress, cfg.ServerPort), handler)
	if err != nil {
		errCh <- err
	}
}

func main() {
	log.SetFlags(log.Ldate | log.LUTC | log.Lmicroseconds | log.Llongfile)

	cfg := ServerConfig{}
	env.ReadToCfg(&cfg)

	// remove trailing slash
	if cfg.AppUrl[len(cfg.AppUrl)-1] == '/' {
		cfg.AppUrl = cfg.AppUrl[:len(cfg.AppUrl)-1]
	}

	regkeysStore, regkeyErrCh, err := rckstrvcache.InitializeStore(time.Minute * 15)
	if err != nil {
		log.Fatalf("fatal error occured while trying to initialize regkey store: %v", err)
	}

	defer func(regkeysStore *rckstrvcache.Store) {
		err := regkeysStore.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(regkeysStore)

	oneTimeAccessTokenStore, oneTimeAccessTokenErrCh, err := rckstrvcache.InitializeStore(time.Minute)
	if err != nil {
		log.Fatalf("fatal error occured while trying to initialize one time access token store: %v", err)
	}

	defer func(oneTimeAccessTokenStore *rckstrvcache.Store) {
		err := oneTimeAccessTokenStore.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(oneTimeAccessTokenStore)

	db, err := sql.Open("sqlite3", cfg.DatabaseUrl)
	if err != nil {
		log.Fatal(err)
	}

	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Println(err)
		}
	}(db)

	err = database.Migrate(db, map[string]string{
		"0001_users": users.MigrationFile,
	})
	if err != nil {
		log.Fatal(err)
	}

	httpServerErrCh := make(chan error)

	go startServer(cfg, regkeysStore, oneTimeAccessTokenStore, db, httpServerErrCh)

	for {
		select {
		case err = <-httpServerErrCh:
			log.Fatalf("Error from http server: %v", err)
		case err = <-oneTimeAccessTokenErrCh:
			log.Fatalf("Error from one time access token store: %v", err)
		case err = <-regkeyErrCh:
			log.Fatalf("Error from regkey store: %v", err)
		default:
			// nothing
		}
	}
}
