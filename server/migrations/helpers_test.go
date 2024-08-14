package migrations

import (
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"testing"
)

func TestDoesTableExists(t *testing.T) {
	t.Parallel()

	t.Run("returns true when table exists", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatal(err)
		}

		defer db.Close()

		_, err = db.Exec("CREATE TABLE sometable (somecolumn int);")
		if err != nil {
			t.Fatal(err)
		}

		exists, err := DoesTableExists(db, "sometable")
		if err != nil {
			t.Fatal(err)
		}

		if !exists {
			t.Error("Expected false, received true.")
		}
	})

	t.Run("returns false when table does not exist", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatal(err)
		}

		defer db.Close()

		exists, err := DoesTableExists(db, "sometable")
		if err != nil {
			t.Fatal(err)
		}

		if exists {
			t.Error("Expected false, received true.")
		}
	})
}

func TestCreateMigrationsTable(t *testing.T) {
	t.Run("succeeds if migrations table does not exist", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatal(err)
		}

		defer db.Close()

		err = createMigrationsTable(db)
		if err != nil {
			t.Fatal(err)
		}

		row := db.QueryRow("SELECT * FROM migrations")

		var id int

		if err = row.Scan(&id); !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("Expected ErrNoRows, received: %v", err)
		}
	})

	t.Run("throws an error if migrations table already exists", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatal(err)
		}

		defer db.Close()

		_, err = db.Exec("CREATE TABLE migrations (id INTEGER PRIMARY KEY);")
		if err != nil {
			t.Fatal(err)
		}

		err = createMigrationsTable(db)
		if !errors.Is(err, ErrMigrationsTableAlreadyExists) {
			t.Errorf("Expected ErrMigrationsTableAlreadyExists error, but received: %v", err)
		}
	})
}

func TestIsMigrationConfirmed(t *testing.T) {
	t.Run("returns true if migration is confirmed", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatal(err)
		}

		defer db.Close()

		err = createMigrationsTable(db)
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.Exec("INSERT INTO migrations (name) VALUES ('somemigration');")
		if err != nil {
			t.Fatal(err)
		}

		result, err := isMigrationConfirmed(db, "somemigration")
		if err != nil {
			t.Fatal(err)
		}

		if !result {
			t.Error("Expected true, received false")
		}
	})

	t.Run("returns false if migration is not confirmed", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatal(err)
		}

		defer db.Close()

		err = createMigrationsTable(db)
		if err != nil {
			t.Fatal(err)
		}

		result, err := isMigrationConfirmed(db, "somemigration")
		if err != nil {
			t.Fatal(err)
		}

		if result != false {
			t.Error("Expected false, received true")
		}
	})

	t.Run("returns ErrMigrationsTableDoesNotExist if migrations table does not exist", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatal(err)
		}

		defer db.Close()

		_, err = isMigrationConfirmed(db, "somemigration")
		if !errors.Is(err, ErrMigrationsTableAlreadyExists) {
			t.Errorf("expected ErrMigrationsTableAlreadyExists, received: %v", err)
		}
	})
}

func TestConfirmMigration(t *testing.T) {
	t.Run("succeeds", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatal(err)
		}

		defer db.Close()

		err = createMigrationsTable(db)
		if err != nil {
			t.Fatal(err)
		}

		err = confirmMigration(db, "somemigration")
		if err != nil {
			t.Errorf("Expected nil, received: %v", err)
		}
	})

	t.Run("returns ErrMigrationsTableDoesNotExist if migrations table does not exist", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatal(err)
		}

		defer db.Close()

		err = confirmMigration(db, "somemigration")
		if !errors.Is(err, ErrMigrationsTableAlreadyExists) {
			t.Errorf("Expected ErrMigrationsTableAlreadyExists, received '%v'", err)
		}
	})
}

func TestMigrate(t *testing.T) {
	// TODO
}
