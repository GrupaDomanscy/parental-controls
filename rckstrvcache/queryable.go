package rckstrvcache

import "database/sql"

type Queryable struct {
	db *sql.DB
	tx *sql.Tx
}

func NewQueryableWithDb(db *sql.DB) *Queryable {
	return &Queryable{
		db: db,
		tx: nil,
	}
}

func NewQueryableWithTx(tx *sql.Tx) *Queryable {
	return &Queryable{
		db: nil,
		tx: tx,
	}
}

func (queryable *Queryable) QueryRow(sql string, args ...interface{}) *sql.Row {
	if queryable.db != nil {
		return queryable.db.QueryRow(sql, args...)
	}

	return queryable.tx.QueryRow(sql, args...)
}

func (queryable *Queryable) Exec(sql string, args ...interface{}) (sql.Result, error) {
	if queryable.db != nil {
		return queryable.db.Exec(sql, args...)
	}

	return queryable.tx.Exec(sql, args...)
}
