package users

import (
	"database/sql"
	"domanscy.group/parental-controls/server/database"
	_ "embed"
	_ "github.com/mattn/go-sqlite3"
	"testing"
)

//go:embed migration.sql
var migrationFile string

func TestFindOneByEmail(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	err = database.Migrate(db, map[string]string{
		"0001_users": migrationFile,
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
		userModel, err = FindOneByEmail(db, "test@localhost.local")
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
		userModel, err = FindOneByEmail(db, "test@localhost.local2")
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
		"0001_users": migrationFile,
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
		userModel, err = FindOneById(db, int(id))
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
		userModel, err = FindOneById(db, int(id)+1)
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
		"0001_users": migrationFile,
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
		userModels, err = GetAllByEmailSearch(db, "est")
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
		userModels, err = GetAllByEmailSearch(db, "est")
		if err != nil {
			t.Fatal(err)
		}

		if len(userModels) != 0 {
			t.Errorf("Expected 0 user models, received: %d", len(userModels))
		}
	})

	t.Run("returns empty array if query does not match any row", func(t *testing.T) {
		var userModels []Model
		userModels, err = GetAllByEmailSearch(db, "olijkhhasdjlksdjagasjkdhasdjhk")
		if err != nil {
			t.Fatal(err)
		}
		if len(userModels) != 0 {
			t.Errorf("Expected 0 user models, received: %d", len(userModels))
		}
	})
}
