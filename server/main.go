package main

import (
	"crypto/rsa"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"domanscy.group/env"
	"domanscy.group/parental-controls/server/database"
	"domanscy.group/parental-controls/server/users"
	"domanscy.group/rckstrvcache"
	"github.com/go-chi/chi"
	_ "github.com/mattn/go-sqlite3"
)

type ServerConfig struct {
	AppUrl        string
	ServerAddress string
	ServerPort    uint16

	EmailFromAddress string
	SmtpAddress      string
	SmtpPort         uint16

	BearerTokenPrivateKey *rsa.PrivateKey

	DatabaseUrl string
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

func readConfig() ServerConfig {
	appUrl, exists, err := env.ParseValidUrlVarWithHttpOrHttpsProtocol("APP_URL")
	if !exists {
		log.Fatalf("env '%s' is required", "APP_URL")
	}

	if err != nil {
		log.Fatalf("env '%s' parsing error: %v", "APP_URL", err)
	}

	appUrlWithoutTrailingSlash := appUrl.String()

	// remove trailing slash
	if appUrlWithoutTrailingSlash[len(appUrlWithoutTrailingSlash)-1] == '/' {
		appUrlWithoutTrailingSlash = appUrlWithoutTrailingSlash[:len(appUrlWithoutTrailingSlash)-1]
	}

	serverAddress, exists := env.ParseStringVar("SERVER_ADDRESS")
	if !exists {
		log.Fatalf("env '%s' is required", "SERVER_ADDRESS")
	}

	serverPort, exists, err := env.ParseUint16Var("SERVER_PORT")
	if !exists {
		log.Fatalf("env '%s' is required", "SERVER_PORT")
	}

	if err != nil {
		log.Fatalf("env '%s' parsing error: %v", "SERVER_PORT", err)
	}

	emailFromAddress, exists := env.ParseStringVar("EMAIL_FROM_ADDRESS")
	if !exists {
		log.Fatalf("env '%s' is required", "EMAIL_FROM_ADDRESS")
	}

	smtpAddress, exists := env.ParseStringVar("SMTP_ADDRESS")
	if !exists {
		log.Fatalf("env '%s' is required", "SMTP_ADDRESS")
	}

	smtpPort, exists, err := env.ParseUint16Var("SMTP_PORT")
	if !exists {
		log.Fatalf("env '%s' is required", "SMTP_PORT")
	}

	if err != nil {
		log.Fatalf("env '%s' parsing error: %v", "SMTP_PORT", err)
	}

	bearerTokenPrivateKey, exists, err := env.ParsePrivateKeyVarFromFilePath("BEARER_TOKEN_PRIVATE_KEY")
	if !exists {
		log.Fatalf("env '%s' is required", "BEARER_TOKEN_PRIVATE_KEY")
	}

	if err != nil {
		log.Fatalf("env '%s' parsing error: %v", "BEARER_TOKEN_PRIVATE_KEY", err)
	}

	databaseUrl, exists := env.ParseStringVar("DATABASE_URL")
	if !exists {
		log.Fatalf("env '%s' is required", "DATABASE_URL")
	}

	cfg := ServerConfig{
		AppUrl:                appUrlWithoutTrailingSlash,
		ServerAddress:         serverAddress,
		ServerPort:            serverPort,
		EmailFromAddress:      emailFromAddress,
		SmtpAddress:           smtpAddress,
		SmtpPort:              smtpPort,
		BearerTokenPrivateKey: bearerTokenPrivateKey,
		DatabaseUrl:           databaseUrl,
	}

	return cfg
}

func logFatalIfErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	log.SetFlags(log.Ldate | log.LUTC | log.Lmicroseconds | log.Llongfile)

	cfg := readConfig()

	regkeysStore, regkeyErrCh, err := rckstrvcache.InitializeStore(time.Minute * 15)
	if err != nil {
		log.Fatalf("fatal error occured while trying to initialize regkey store: %v", err)
	}

	defer func(store *rckstrvcache.Store) {
		logFatalIfErr(store.Close())
	}(regkeysStore)

	otatStore, otatStoreErrCh, err := rckstrvcache.InitializeStore(time.Minute)
	if err != nil {
		log.Fatalf("fatal error occured while trying to initialize one time access token store: %v", err)
	}

	defer func(store *rckstrvcache.Store) {
		logFatalIfErr(store.Close())
	}(otatStore)

	db, err := sql.Open("sqlite3", cfg.DatabaseUrl)
	if err != nil {
		log.Fatal(err)
	}

	defer func(db *sql.DB) {
		logFatalIfErr(db.Close())
	}(db)

	err = database.Migrate(db, map[string]string{
		"0001_users": users.MigrationFile,
	})
	if err != nil {
		log.Fatal(err)
	}

	httpServerErrCh := make(chan error)

	go startServer(cfg, regkeysStore, otatStore, db, httpServerErrCh)

	for {
		select {
		case err = <-httpServerErrCh:
			log.Fatalf("Error from http server: %v", err)
		case err = <-otatStoreErrCh:
			log.Fatalf("Error from one time access token store: %v", err)
		case err = <-regkeyErrCh:
			log.Fatalf("Error from regkey store: %v", err)
		default:
			// nothing
		}
	}
}
