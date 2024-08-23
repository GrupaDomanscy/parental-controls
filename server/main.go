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
	appUrl, exists, err := env.ParseValidUrlWithHttpOrHttpsProtocol("APP_URL")
	if !exists {
		log.Fatalf("env '%s' is required", "APP_URL")
	}

	if err != nil {
		log.Fatalf("env '%s' parsing error: %v", err)
	}

	appUrlWithoutTrailingSlash := appUrl.String()

	// remove trailing slash
	if appUrlWithoutTrailingSlash[len(appUrlWithoutTrailingSlash)-1] == '/' {
		appUrlWithoutTrailingSlash = appUrlWithoutTrailingSlash[:len(appUrlWithoutTrailingSlash)-1]
	}

	serverAddress, exists := env.ParseStringVar("SERVER_ADDRESS")
	if !exists {
		log.Fatalf("env '%s' is required", "APP_URL")
	}

	serverPort, exists, err := env.ParseUint16Var("SERVER_PORT")
	if !exists {
		log.Fatalf("env '%s' is required", "SERVER_PORT")
	}

	if err != nil {
		log.Fatalf("env '%s' parsing error: %v", err)
	}

	emailFromAddress, exists := env.ParseStringVar("EMAIL_FROM_ADDRESS")
	if !exists {
		log.Fatalf("env '%s' is required", "EMAIL_FROM_ADDRESS")
	}

	smtpAddress, exists, err := env.ParseUrlVarOnlyWithHost("SMTP_ADDRESS")
	if !exists {
		log.Fatalf("env '%s' is required", "EMAIL_FROM_ADDRESS")
	}

	if err != nil {
		log.Fatalf("env '%s' parsing error: %v", err)
	}

	smtpPort, exists, err := env.ParseUint16Var("SMTP_PORT")
	if !exists {
		log.Fatalf("env '%s' is required", "SMTP_PORT")
	}

	if err != nil {
		log.Fatalf("env '%s' parsing error: %v", err)
	}

	bearerTokenPrivateKey, exists, err := env.ParsePrivateKeyVarFromFilePath("BEARER_TOKEN_PRIVATE_KEY")
	if !exists {
		log.Fatalf("env '%s' is required", "BEARER_TOKEN_PRIVATE_KEY")
	}

	if err != nil {
		log.Fatalf("env '%s' parsing error: %v", err)
	}

	databaseUrl, exists := env.ParseStringVar("DATABASE_URL")
	if !exists {
		log.Fatalf("env '%s' is required", "BEARER_TOKEN_PRIVATE_KEY")
	}

	cfg := ServerConfig{
		AppUrl:                appUrlWithoutTrailingSlash,
		ServerAddress:         serverAddress,
		ServerPort:            serverPort,
		EmailFromAddress:      emailFromAddress,
		SmtpAddress:           smtpAddress.String(),
		SmtpPort:              smtpPort,
		BearerTokenPrivateKey: bearerTokenPrivateKey,
		DatabaseUrl:           databaseUrl,
	}

	return cfg
}

func main() {
	log.SetFlags(log.Ldate | log.LUTC | log.Lmicroseconds | log.Llongfile)

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
