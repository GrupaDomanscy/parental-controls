package rckstrvcache

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"time"
)

type StoreCompatible interface {
	Get(key string) (value string, exists bool, err error)
	Put(value string) (key string, err error)
	Delete(key string) (affected bool, err error)
}

type Store struct {
	StoreCompatible

	tempFile  *os.File
	ctx       context.Context
	db        *sql.DB
	cancelCtx func()
	ttl       int64
}

func (store *Store) deleteRoutine(errCh chan<- error) {
	for {
		select {
		case <-store.ctx.Done():
			return
		case <-time.After(time.Millisecond * 1000): // dear god, don't kill that database. 1 sec is minimum
			tx, err := store.db.BeginTx(store.ctx, nil)
			if err != nil {
				errCh <- errors.Join(err, fmt.Errorf("an error occured while trying to open the transaction: %w", err))
				return
			}

			_, err = tx.Exec("DELETE FROM data WHERE delete_at < ?", time.Now().UnixMilli())
			if err != nil {
				rollbackErr := tx.Rollback()
				if rollbackErr != nil {
					err = errors.Join(err, fmt.Errorf("an error occured while trying to delete expired values: %w", err))
				}

				errCh <- err
				return
			}

			err = tx.Commit()
			if err != nil {
				errCh <- err
			}
		}
	}
}

var ErrTTLCannotBeShorterThan1Sec = errors.New("ttl can not be shorter than 1 second")

func InitializeStore(ttl time.Duration) (*Store, <-chan error, error) {
	if ttl.Milliseconds() < 1000 {
		return nil, nil, ErrTTLCannotBeShorterThan1Sec
	}

	ctx, cancelCtx := context.WithCancel(context.Background())

	file, err := os.CreateTemp("", "rckstrvcache")
	if err != nil {
		cancelCtx()
		return nil, nil, err
	}

	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared", file.Name()))
	if err != nil {
		cancelCtx()
		return nil, nil, fmt.Errorf("error occured while trying to open the database connection: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		cancelCtx()
		return nil, nil, fmt.Errorf("error occured while trying to enable WAL mode: %w", err)
	}

	_, err = db.Exec("CREATE TABLE data (key VARCHAR PRIMARY KEY, value VARCHAR NOT NULL, delete_at INTEGER NOT NULL);")
	if err != nil {
		cancelCtx()
		return nil, nil, fmt.Errorf("error occured while trying to create data table in db: %w", err)
	}

	errCh := make(chan error)

	store := &Store{
		tempFile:  file,
		ctx:       ctx,
		db:        db,
		cancelCtx: cancelCtx,
		ttl:       ttl.Milliseconds(),
	}

	go store.deleteRoutine(errCh)

	return store, errCh, nil
}

func (store *Store) Close() error {
	store.cancelCtx()

	dbErr := store.db.Close()
	fileErr := store.tempFile.Close()

	if dbErr != nil && fileErr != nil {
		return errors.Join(dbErr, fileErr)
	} else if dbErr != nil {
		return dbErr
	} else if fileErr != nil {
		return fileErr
	} else {
		return nil
	}
}

func (store *Store) GetAllKeys() ([]string, error) {
	rows, err := store.db.Query("SELECT `key` FROM data")
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0)

	for rows.Next() {
		var key string

		err := rows.Scan(&key)
		if err != nil {
			return nil, err
		}

		keys = append(keys, key)
	}

	return keys, nil
}

func (store *Store) Get(key string) (value string, exists bool, err error) {
	queryable := NewQueryableWithDb(store.db)

	return getFromDb(queryable, key)
}

func (store *Store) Put(value string) (key string, err error) {
	queryable := NewQueryableWithDb(store.db)

	return putAndGenerateRandomKeyForValue(queryable, value, store.ttl)
}

func (store *Store) Delete(key string) (affected bool, err error) {
	queryable := NewQueryableWithDb(store.db)

	return deleteFromDb(queryable, key)
}

type StoreInTx struct {
	tx  *sql.Tx
	ttl int64
}

func (storeInTx *StoreInTx) Get(key string) (value string, exists bool, err error) {
	queryable := NewQueryableWithTx(storeInTx.tx)

	return getFromDb(queryable, key)
}

func (storeInTx *StoreInTx) Put(value string) (key string, err error) {
	queryable := NewQueryableWithTx(storeInTx.tx)

	return putAndGenerateRandomKeyForValue(queryable, value, storeInTx.ttl)
}

func (storeInTx *StoreInTx) Delete(key string) (affected bool, err error) {
	queryable := NewQueryableWithTx(storeInTx.tx)

	return deleteFromDb(queryable, key)
}

type InTransactionCallback = func(store StoreCompatible) error

func (store *Store) InTransaction(callback InTransactionCallback) (err error) {
	tx, err := store.db.BeginTx(store.ctx, nil)
	if err != nil {
		return fmt.Errorf("error occured while trying to open transaction: %w", err)
	}

	storeInTx := &StoreInTx{tx: tx, ttl: store.ttl}

	err = callback(storeInTx)
	if err != nil {
		txErr := tx.Rollback()
		if txErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to rollback transaction: %v", err))
		}

		return
	}

	return tx.Commit()
}
