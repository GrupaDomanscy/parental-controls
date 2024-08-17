package users

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

var ErrEmailCannotBeEmpty = errors.New("email can not be empty")
var ErrUserWithGivenEmailAlreadyExists = errors.New("user with given email already exists")

type Model struct {
	Id        int
	Email     string
	CreatedAt time.Time
}

func FindOneById(db *sql.DB, id int) (*Model, error) {
	row := db.QueryRow("SELECT id, email, created_at FROM users WHERE id = $1", id)

	user := &Model{}

	err := row.Scan(&user.Id, &user.Email, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("error occured while trying to scan the row for values: %w", err)
	}

	return user, nil
}

func FindOneByEmail(db *sql.DB, email string) (*Model, error) {
	row := db.QueryRow("SELECT * FROM users WHERE email = $1", email)

	user := &Model{}

	err := row.Scan(&user.Id, &user.Email, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("error occured while trying to scan the row for values: %w", err)
	}

	return user, nil
}

func GetAllByEmailSearch(db *sql.DB, emailPart string) ([]Model, error) {
	if len(emailPart) == 0 {
		return []Model{}, nil
	}

	emailPart = strings.Join(strings.Split(emailPart, "_"), "\\_")
	emailPart = strings.Join(strings.Split(emailPart, "%"), "\\%")

	queryParam := "%" + emailPart + "%"

	rows, err := db.Query("SELECT * FROM users WHERE email LIKE $1", queryParam)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query 'SELECT * FROM users ...': %w", err)
	}

	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("error occured while trying to close rows reader: %v", err)
		}
	}(rows)

	var users []Model

	for rows.Next() {
		user := Model{}
		err := rows.Scan(&user.Id, &user.Email, &user.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan the row for values: %w", err)
		}

		users = append(users, user)
	}

	return users, nil
}

func Create(db *sql.DB, email string) (int, error) {
	if email == "" {
		return 0, ErrEmailCannotBeEmpty
	}

	exec, err := db.Exec("INSERT INTO users (email) VALUES (?);", email)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: users.email" {
			return 0, ErrUserWithGivenEmailAlreadyExists
		}

		return 0, fmt.Errorf("an error occured while trying to execute query 'INSERT INTO users ...': %w", err)
	}

	id, err := exec.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("an error occured while trying to get last inserted id from database: %w", err)
	}

	return int(id), nil
}

var ErrUserWithThisIdDoesNotExist = errors.New("user with this id does not exist")

func Update(db *sql.DB, id int, newEmail string) error {
	executed, err := db.Exec("UPDATE users SET email = ? WHERE id = ?", newEmail, id)
	if err != nil {
		return err
	}

	affectedRows, err := executed.RowsAffected()
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: users.email" {
			return ErrUserWithGivenEmailAlreadyExists
		}

		return fmt.Errorf("unknown error occured when trying to get number of affected rows: %w", err)
	}

	if affectedRows == 0 {
		return ErrUserWithThisIdDoesNotExist
	} else if affectedRows >= 2 {
		return fmt.Errorf("affected rows is equal to: %d, this is a database error", affectedRows)
	}

	return nil
}
