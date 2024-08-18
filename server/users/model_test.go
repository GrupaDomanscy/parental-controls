package users

import (
	"database/sql"
	"domanscy.group/parental-controls/server/database"
	_ "embed"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"testing"
)

func TestFindOneByEmail(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	err = database.Migrate(db, map[string]string{
		"0001_users": MigrationFile,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec("INSERT INTO users (email) VALUES (?);", "test@localhost.local")
	if err != nil {
		t.Fatal(err)
	}

	t.Parallel()

	t.Run("returns some user if user with provided email exists", func(t *testing.T) {
		userModel := &Model{}

		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			err = tx.Commit()
			if err != nil {
				t.Fatal(err)
			}
		}()

		userModel, err = FindOneByEmail(tx, "test@localhost.local")
		if err != nil {
			t.Fatal(err)
		}

		if userModel.Id <= 0 {
			t.Errorf("Expected id to be bigger than 0. Received: %d.", userModel.Id)
		}

		if userModel.Email != "test@localhost.local" {
			t.Errorf("Expected email to be equal to 'test@localhost.local', received '%s'.", userModel.Email)
		}

		if userModel.CreatedAt.UnixMilli() == 0 {
			t.Errorf("Expected created_at to not be equal to 0, received %d.", userModel.CreatedAt.UnixMilli())
		}
	})

	t.Run("returns nil if user with provided email does not exist", func(t *testing.T) {
		userModel := &Model{}

		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			err = tx.Commit()
			if err != nil {
				t.Fatal(err)
			}
		}()

		userModel, err = FindOneByEmail(tx, "test@localhost.local2")
		if err != nil {
			t.Fatal(err)
		}

		if userModel != nil {
			t.Errorf("Expected nil, received: %v", userModel)
		}
	})
}

func TestFindOneById(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	err = database.Migrate(db, map[string]string{
		"0001_users": MigrationFile,
	})
	if err != nil {
		t.Fatal(err)
	}

	info, err := db.Exec("INSERT INTO users (email) VALUES (?);", "test@localhost.local")
	if err != nil {
		t.Fatal(err)
	}

	id, err := info.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	t.Parallel()

	t.Run("returns some user if user with provided id exists", func(t *testing.T) {
		userModel := &Model{}

		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			err = tx.Commit()
			if err != nil {
				t.Fatal(err)
			}
		}()

		userModel, err = FindOneById(tx, int(id))
		if err != nil {
			t.Fatal(err)
		}

		if userModel.Id != int(id) {
			t.Errorf("Expected id to be %d. Received: %d.", id, userModel.Id)
		}

		if userModel.Email != "test@localhost.local" {
			t.Errorf("Expected email to be equal to 'test@localhost.local', received '%s'.", userModel.Email)
		}

		if userModel.CreatedAt.UnixMilli() == 0 {
			t.Errorf("Expected created_at to not be equal to 0, received %d.", userModel.CreatedAt.UnixMilli())
		}
	})

	t.Run("returns nil if user with provided email does not exist", func(t *testing.T) {
		userModel := &Model{}

		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			err = tx.Commit()
			if err != nil {
				t.Fatal(err)
			}
		}()

		userModel, err = FindOneById(tx, int(id)+1)
		if err != nil {
			t.Fatal(err)
		}

		if userModel != nil {
			t.Errorf("Expected nil, received: %v", userModel)
		}
	})
}

func TestGetAllByEmailSearch(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	err = database.Migrate(db, map[string]string{
		"0001_users": MigrationFile,
	})
	if err != nil {
		t.Fatal(err)
	}

	var ids []int

	for email := range []string{"test@localhost.local", "test2@localhost.local", "hoho@sss.local"} {
		info, err := db.Exec("INSERT INTO users (email) VALUES (?);", email)
		if err != nil {
			t.Fatal(err)
		}

		id, err := info.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}

		ids = append(ids, int(id))
	}

	t.Parallel()

	t.Run("returns multiple results if query matches multiple rows", func(t *testing.T) {
		var userModels []Model

		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			err = tx.Commit()
			if err != nil {
				t.Fatal(err)
			}
		}()

		userModels, err = GetAllByEmailSearch(tx, "est")
		if err != nil {
			t.Fatal(err)
		}

		for i, userModel := range userModels {
			if userModel.Id != ids[i] {
				t.Errorf("Expected id to be %d. Received: %d.", ids[i], userModel.Id)
			}

			if userModel.Email != "test@localhost.local" && userModel.Email != "test@localhost.local2" {
				t.Errorf("Expected email to be equal to 'test@localhost.local' or 'test@localhost.local2', received '%s'.", userModel.Email)
			}

			if userModel.CreatedAt.UnixMilli() == 0 {
				t.Errorf("Expected created_at to not be equal to 0, received %d.", userModel.CreatedAt.UnixMilli())
			}
		}
	})

	t.Run("returns nothing if query is empty", func(t *testing.T) {
		var userModels []Model

		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			err = tx.Commit()
			if err != nil {
				t.Fatal(err)
			}
		}()

		userModels, err = GetAllByEmailSearch(tx, "est")
		if err != nil {
			t.Fatal(err)
		}

		if len(userModels) != 0 {
			t.Errorf("Expected 0 user models, received: %d", len(userModels))
		}
	})

	t.Run("returns empty array if query does not match any row", func(t *testing.T) {
		var userModels []Model

		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			err = tx.Commit()
			if err != nil {
				t.Fatal(err)
			}
		}()

		userModels, err = GetAllByEmailSearch(tx, "olijkhhasdjlksdjagasjkdhasdjhk")
		if err != nil {
			t.Fatal(err)
		}
		if len(userModels) != 0 {
			t.Errorf("Expected 0 user models, received: %d", len(userModels))
		}
	})
}

func TestCreate(t *testing.T) {
	userModel := &Model{
		Email: "hello@world.local",
	}

	t.Run("returns error if email is empty", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatal(err)
		}

		err = database.Migrate(db, map[string]string{
			"0001_users": MigrationFile,
		})
		if err != nil {
			t.Fatal(err)
		}

		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			err = tx.Commit()
			if err != nil {
				t.Fatal(err)
			}
		}()

		_, err = Create(tx, "")
		if !errors.Is(err, ErrEmailCannotBeEmpty) {
			t.Fatal(err)
		}

		row := tx.QueryRow("SELECT COUNT(*) FROM users")
		var rows int

		err = row.Scan(&rows)
		if err != nil {
			t.Fatal(err)
		}

		if rows != 0 {
			t.Errorf("Expected rows to be equal to 0.")
		}
	})

	t.Run("succeeds and returns valid id", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatal(err)
		}

		err = database.Migrate(db, map[string]string{
			"0001_users": MigrationFile,
		})
		if err != nil {
			t.Fatal(err)
		}

		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			err = tx.Commit()
			if err != nil {
				t.Fatal(err)
			}
		}()

		id, err := Create(tx, userModel.Email)
		if err != nil {
			t.Fatal(err)
		}

		if id <= 0 {
			t.Errorf("Expected id to be bigger than 0, received %d.", id)
		}

		row := tx.QueryRow("SELECT id, email, created_at FROM users WHERE id = ?", id)
		var receivedEmail string

		err = row.Scan(&userModel.Id, &receivedEmail, &userModel.CreatedAt)
		if err != nil {
			t.Fatal(err)
		}

		if userModel.Id != id {
			t.Errorf("Expected id to be '%d', received '%d'.", id, userModel.Id)
		}

		if userModel.Email != receivedEmail {
			t.Errorf("Expected email to be '%s', but received '%s'", userModel.Email, receivedEmail)
		}

		if userModel.CreatedAt.UnixMilli() == 0 {
			t.Errorf("Expected created at to not be equal to 0.")
		}

		row = tx.QueryRow("SELECT COUNT(*) FROM users")
		var rows int

		err = row.Scan(&rows)
		if err != nil {
			t.Fatal(err)
		}

		if rows != 1 {
			t.Errorf("Expected rows to be equal to 1.")
		}
	})

	t.Run("throws error when user with given email already exists", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatal(err)
		}

		err = database.Migrate(db, map[string]string{
			"0001_users": MigrationFile,
		})
		if err != nil {
			t.Fatal(err)
		}

		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			err = tx.Commit()
			if err != nil {
				t.Fatal(err)
			}
		}()

		_, err = Create(tx, userModel.Email)
		if err != nil {
			t.Fatal(err)
		}

		_, err = Create(tx, userModel.Email)
		if !errors.Is(err, ErrUserWithGivenEmailAlreadyExists) {
			t.Errorf("Expected ErrUserWithGivenEmailAlreadyExists, received: %v", err)
		}

		row := tx.QueryRow("SELECT COUNT(*) FROM users")
		var rows int

		err = row.Scan(&rows)
		if err != nil {
			t.Fatal(err)
		}

		if rows != 1 {
			t.Errorf("Expected rows to be equal to 1.")
		}
	})

}

func TestUpdate(t *testing.T) {
	t.Run("succeeds when everything is ok", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatal(err)
		}

		err = database.Migrate(db, map[string]string{
			"0001_users": MigrationFile,
		})
		if err != nil {
			t.Fatal(err)
		}

		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			err = tx.Commit()
			if err != nil {
				t.Fatal(err)
			}
		}()

		// Insert a user to update
		id, err := Create(tx, "oldemail@domain.com")
		if err != nil {
			t.Fatal(err)
		}

		newEmail := "newemail@domain.com"
		err = Update(tx, id, newEmail)
		if err != nil {
			t.Fatal(err)
		}

		// Verify the email is updated
		row := tx.QueryRow("SELECT email FROM users WHERE id = ?", id)
		var receivedEmail string
		err = row.Scan(&receivedEmail)
		if err != nil {
			t.Fatal(err)
		}

		if receivedEmail != newEmail {
			t.Errorf("Expected email to be '%s', but received '%s'", newEmail, receivedEmail)
		}
	})

	t.Run("returns ErrUserWithThisIdDoesNotExist when user with given id does not exist", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatal(err)
		}

		err = database.Migrate(db, map[string]string{
			"0001_users": MigrationFile,
		})
		if err != nil {
			t.Fatal(err)
		}

		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			err = tx.Commit()
			if err != nil {
				t.Fatal(err)
			}
		}()

		nonExistentId := 9999
		err = Update(tx, nonExistentId, "email@domain.com")
		if !errors.Is(err, ErrUserWithThisIdDoesNotExist) {
			t.Errorf("Expected ErrUserWithThisIdDoesNotExist, received: %v", err)
		}
	})

	t.Run("returns ErrUserWithGivenEmailAlreadyExists when user with given newEmail already exists", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatal(err)
		}

		err = database.Migrate(db, map[string]string{
			"0001_users": MigrationFile,
		})
		if err != nil {
			t.Fatal(err)
		}

		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			err = tx.Commit()
			if err != nil {
				t.Fatal(err)
			}
		}()

		// Create two users
		_, err = Create(tx, "user1@domain.com")
		if err != nil {
			t.Fatal(err)
		}

		id2, err := Create(tx, "user2@domain.com")
		if err != nil {
			t.Fatal(err)
		}

		// Try updating the second user with an email that already exists
		err = Update(tx, id2, "user1@domain.com")
		if !errors.Is(err, ErrUserWithGivenEmailAlreadyExists) {
			t.Errorf("Expected ErrUserWithGivenEmailAlreadyExists, received: %v", err)
		}
	})
}
