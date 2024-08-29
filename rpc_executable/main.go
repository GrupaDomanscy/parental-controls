package rpc_executable

import (
	"database/sql"
	"log"

	"domanscy.group/parental-controls/server/database"
	"domanscy.group/parental-controls/server/users"
	"domanscy.group/rckstrvcache"
	server "domanscy.group/server"
)

func main() {
	log.SetFlags(log.Ldate | log.LUTC | log.Lmicroseconds | log.Llongfile)

	cfg := server.readConfig()

	regkeysStore, regkeyErrCh, err := rckstrvcache.InitializeStore(time.Minute * 15)
	if err != nil {
		log.Fatalf("fatal error occured while trying to initialize regkey store: %v", err)
	}

	defer func(store *rckstrvcache.Store) {
		server.logFatalIfErr(store.Close())
	}(regkeysStore)

	otatStore, otatStoreErrCh, err := rckstrvcache.InitializeStore(time.Minute)
	if err != nil {
		log.Fatalf("fatal error occured while trying to initialize one time access token store: %v", err)
	}

	defer func(store *rckstrvcache.Store) {
		server.logFatalIfErr(store.Close())
	}(otatStore)

	db, err := sql.Open("sqlite3", cfg.DatabaseUrl)
	if err != nil {
		log.Fatal(err)
	}

	defer func(db *sql.DB) {
		server.logFatalIfErr(db.Close())
	}(db)

	err = database.Migrate(db, map[string]string{
		"0001_users": users.MigrationFile,
	})
	if err != nil {
		log.Fatal(err)
	}

	httpServerErrCh := make(chan error)

	go server.StartServer(cfg, regkeysStore, otatStore, db, httpServerErrCh)

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
