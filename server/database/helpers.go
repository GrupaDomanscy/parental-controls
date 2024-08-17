package database

import (
	"database/sql"
	"errors"
	"fmt"
)

var ErrMigrationsTableAlreadyExists = errors.New("migrations table already exists")

func DoesTableExists(db *sql.DB, tableName string) (bool, error) {
	row := db.QueryRow(
		"SELECT name FROM sqlite_schema WHERE type = 'table' AND name = ? AND tbl_name = ? AND sql LIKE 'CREATE TABLE %';",
		tableName,
		tableName,
	)

	var name string

	err := row.Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func createMigrationsTable(db *sql.DB) error {
	_, err := db.Exec("CREATE TABLE migrations (id INTEGER PRIMARY KEY, name VARCHAR, migrated_at INTEGER);")
	if err != nil {
		if err.Error() == "table migrations already exists" {
			return errors.Join(ErrMigrationsTableAlreadyExists, err)
		}
		return fmt.Errorf("an error occured while trying to execute 'create table migrations...': %w", err)
	}

	return nil
}

func isMigrationConfirmed(db *sql.DB, migrationName string) (bool, error) {
	row := db.QueryRow("SELECT id FROM migrations WHERE name = ?;", migrationName)

	var id int

	err := row.Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		if err.Error() == "no such table: migrations" {
			return false, ErrMigrationsTableAlreadyExists
		}

		return false, fmt.Errorf("error occured while trying to run sql query 'SELECT id FROM migrations...': %w", err)
	}

	return true, nil
}

func confirmMigration(db *sql.DB, migrationName string) error {
	_, err := db.Exec("INSERT INTO migrations (name) VALUES (?);", migrationName)
	if err != nil {
		if err.Error() == "no such table: migrations" {
			return ErrMigrationsTableAlreadyExists
		}

		return fmt.Errorf("failed to run sql 'INSERT INTO migrations...': %w", err)
	}

	return nil
}

func Migrate(db *sql.DB, migrations map[string]string) error {
	migrationsTableExists, err := DoesTableExists(db, "migrations")
	if err != nil {
		return fmt.Errorf("error occured while trying to check if migrations table exists: %w", err)
	}

	if !migrationsTableExists {
		err = createMigrationsTable(db)
		if err != nil {
			return fmt.Errorf("error occured while trying to create migrations table: %w", err)
		}
	}

	for migrationName, sqlQuery := range migrations {
		migrationConfirmed, err := isMigrationConfirmed(db, migrationName)
		if err != nil {
			return fmt.Errorf("error occured while trying to obtain information about confirmation of migration execution: %w", err)
		}

		if migrationConfirmed {
			continue
		}

		_, err = db.Exec(sqlQuery)
		if err != nil {
			return fmt.Errorf("error occured while trying to execute migration '%s' with sql query '%s': %w", migrationName, sqlQuery, err)
		}

		err = confirmMigration(db, migrationName)
		if err != nil {
			return fmt.Errorf("error occured while trying to confirm migration '%s': %w", migrationName, err)
		}
	}

	return nil
}
