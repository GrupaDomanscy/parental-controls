package users

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

type Model struct {
	Id        int
	Email     string
	CreatedAt time.Time
}

func FindOneById(db *sql.DB, id int) (*Model, error) {
	row := db.QueryRow("SELECT id, email, created_at FROM users WHERE id = $1", id)

	user := &Model{}

	err := row.Scan(&user.Id, &user.Email, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("error occured while trying to scan the row for values: %w", err)
	}

	return user, nil
}

func FindOneByEmail(db *sql.DB, email string) (*Model, error) {
	row := db.QueryRow("SELECT * FROM users WHERE email = $1", email)

	user := &Model{}

	err := row.Scan(&user.Id, &user.Email, &user.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("error occured while trying to scan the row for values: %w", err)
	}

	return user, nil
}

func GetAllByEmailSearch(db *sql.DB, emailPart string) ([]Model, error) {
	emailPart = strings.Join(strings.Split(emailPart, "_"), "\\_")
	emailPart = strings.Join(strings.Split(emailPart, "%"), "\\%")

	queryParam := "%" + emailPart + "%"

	rows, err := db.Query("SELECT * FROM users WHERE email LIKE $1", queryParam)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query 'SELECT * FROM users ...': %w", queryParam)
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
