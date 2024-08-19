package rckstrvcache

import (
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"errors"
	"fmt"
	"time"
)

func generateRandomString(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("an unknown error occured while trying to generate random bytes using crypto/rand.Read: %w", err)
	}
	return base32.StdEncoding.EncodeToString(b), nil
}

func putAndGenerateRandomKeyForValue(queryable *Queryable, value string, ttl int64) (string, error) {
	key, err := generateRandomString(128)
	if err != nil {
		return "", fmt.Errorf("an unknown error occured while trying to generate a random key: %w", err)
	}

	_, err = queryable.Exec("INSERT INTO data (key, value, delete_at) VALUES (?, ?, ?)", key, value, time.Now().UnixMilli()+ttl)
	if err != nil {
		return "", err
	}

	return key, nil
}

func getFromDb(queryable *Queryable, key string) (value string, exists bool, err error) {
	row := queryable.QueryRow("SELECT `value` FROM data WHERE `key` = ?", key)

	err = row.Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	} else if err != nil {
		return "", false, err
	}

	return value, true, nil
}

var ErrTooMuchDataDeleted = errors.New("too much data has been deleted, expected 1 row, received more than 1")

func deleteFromDb(queryable *Queryable, key string) (affected bool, err error) {
	executed, err := queryable.Exec("DELETE FROM data WHERE `key` = ?", key)
	if err != nil {
		return false, fmt.Errorf("error occured while trying to execute DELETE sql query: %w", err)
	}

	rowsAffected, err := executed.RowsAffected()
	if err != nil {
		return false, err
	}

	if rowsAffected == 1 {
		return true, nil
	} else if rowsAffected == 0 {
		return false, nil
	} else {
		return false, ErrTooMuchDataDeleted
	}
}
