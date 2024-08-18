package main

import (
	"bytes"
	"database/sql"
	"domanscy.group/parental-controls/server/database"
	"domanscy.group/parental-controls/server/users"
	"encoding/json"
	"log"
	"mailpitsuite"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var testingCfg = &ServerConfig{
	ServerAddress:    "127.0.0.1",
	ServerPort:       8080,
	EmailFromAddress: "test@parental-controls.local",
	SmtpAddress:      "127.0.0.1",
	SmtpPort:         1025,
	DatabaseUrl:      ":memory:",
}

func openDatabase(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", testingCfg.DatabaseUrl)
	if err != nil {
		t.Fatal(err)
	}

	err = database.Migrate(db, map[string]string{
		"0001_users": users.MigrationFile,
	})
	if err != nil {
		return nil
	}

	return db
}

func assertMailpitInboxIsEmpty(t *testing.T, mailpit *mailpitsuite.Api) {
	messages, err := mailpit.GetAllMessages()
	if err != nil {
		t.Fatalf("failed to get all mailpit messages: %s", err.Error())
	}

	if len(messages) != 0 {
		t.Errorf("Length of messages should be equal to 0, received %d", len(messages))
	}
}

func convertStructToJson(t *testing.T, obj interface{}) []byte {
	result, err := json.Marshal(obj)

	if err != nil {
		t.Fatal(err)
	}

	return result
}

func initializeMailpitAndDeleteAllMessages(t *testing.T) *mailpitsuite.Api {
	mailpit, err := mailpitsuite.NewApi(mailpitExeFilePath)
	if err != nil {
		t.Fatal(err)
	}

	err = mailpit.DeleteAllMessages()
	if err != nil {
		t.Fatalf("failed to delete all mailpit messages: %s", err.Error())
	}

	return mailpit
}

func TestHttpAuthLogin(t *testing.T) {
	t.Run("returns 400 if json payload is invalid", func(t *testing.T) {
		t.Parallel()
		mailpit := initializeMailpitAndDeleteAllMessages(t)
		defer mailpit.Close()

		db := openDatabase(t)

		bodyReader := strings.NewReader("notavalidjson")

		recorder := httptest.NewRecorder()
		request, err := http.NewRequest("GET", "http://localhost:8080/login", bodyReader)
		if err != nil {
			t.Fatal(err)
		}

		request.RemoteAddr = "127.0.0.1:51789"

		HttpAuthLogin(testingCfg, db)(recorder, request)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Got %d, want %d", recorder.Code, http.StatusBadRequest)
		}

		if recorder.Body.String() != ErrInvalidJsonPayload.Error() {
			t.Errorf("Got %s, want %s", recorder.Body.String(), ErrInvalidJsonPayload.Error())
		}

		assertMailpitInboxIsEmpty(t, mailpit)
	})

	t.Run("returns 400 if email is invalid", func(t *testing.T) {
		t.Parallel()
		mailpit := initializeMailpitAndDeleteAllMessages(t)
		defer mailpit.Close()

		db := openDatabase(t)

		bodyReader := bytes.NewReader(convertStructToJson(t, struct {
			Email    string `json:"email"`
			Callback string `json:"callback"`
		}{
			Email:    "invalid+.,nbav']@email.local",
			Callback: "http://localhost:8080",
		}))
		recorder := httptest.NewRecorder()

		request, err := http.NewRequest("GET", "http://localhost:8080/login", bodyReader)
		if err != nil {
			t.Fatal(err)
		}

		request.RemoteAddr = "127.0.0.1:51789"

		HttpAuthLogin(testingCfg, db)(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Got %d, want %d", recorder.Code, http.StatusBadRequest)
		}

		if recorder.Body.String() != ErrInvalidEmail.Error() {
			t.Errorf("Got %s, want %s", recorder.Body.String(), ErrInvalidEmail.Error())
		}

		assertMailpitInboxIsEmpty(t, mailpit)
	})

	t.Run("returns 400 if callback is invalid", func(t *testing.T) {
		t.Parallel()
		mailpit := initializeMailpitAndDeleteAllMessages(t)
		defer mailpit.Close()

		db := openDatabase(t)
		bodyReader := bytes.NewReader(convertStructToJson(t, struct {
			Email    string `json:"email"`
			Callback string `json:"callback"`
		}{
			Email:    "hello@world.local",
			Callback: "p;789y124q6tyol789uioy7yui828u90ipriogp[r",
		}))

		recorder := httptest.NewRecorder()

		request, err := http.NewRequest("GET", "http://localhost:8080/login", bodyReader)
		if err != nil {
			t.Fatal(err)
		}

		request.RemoteAddr = "127.0.0.1:51789"

		HttpAuthLogin(testingCfg, db)(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Got %d, want %d", recorder.Code, http.StatusBadRequest)
		}

		if recorder.Body.String() != ErrInvalidCallbackUrl.Error() {
			t.Errorf("Got %s, want %s", recorder.Body.String(), ErrInvalidCallbackUrl.Error())
		}

		assertMailpitInboxIsEmpty(t, mailpit)
	})

	t.Run("returns ErrUserWithGivenEmailDoesNotExist when user with given email does not exist", func(t *testing.T) {
		t.Parallel()
		mailpit := initializeMailpitAndDeleteAllMessages(t)
		defer mailpit.Close()

		db := openDatabase(t)
		bodyReader := bytes.NewReader(convertStructToJson(t, struct {
			Email    string `json:"email"`
			Callback string `json:"callback"`
		}{
			Email:    "hello@world.local",
			Callback: "https://localhost",
		}))

		recorder := httptest.NewRecorder()

		request, err := http.NewRequest("GET", "http://localhost:8080/login", bodyReader)
		if err != nil {
			t.Fatal(err)
		}

		request.RemoteAddr = "127.0.0.1:51789"

		HttpAuthLogin(testingCfg, db)(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Got %d, want %d", recorder.Code, http.StatusBadRequest)
		}

		if recorder.Body.String() != ErrUserWithGivenEmailDoesNotExist.Error() {
			t.Errorf("Got %s, want %s", recorder.Body.String(), ErrInvalidCallbackUrl.Error())
		}

		assertMailpitInboxIsEmpty(t, mailpit)
	})
}

func TestHttpAuthStartRegistrationProcess(t *testing.T) {
	t.Parallel()

	t.Run("returns 400 with ErrInvalidCallbackUrl when callback is invalid", func(t *testing.T) {
		t.Parallel()
		mailpit := initializeMailpitAndDeleteAllMessages(t)
		defer func(mailpit *mailpitsuite.Api) {
			err := mailpit.Close()
			if err != nil {
				log.Println(err)
			}
		}(mailpit)

		db := openDatabase(t)
		bodyReader := bytes.NewReader(convertStructToJson(t, struct {
			Email    string `json:"email"`
			Callback string `json:"callback"`
		}{
			Email:    "hello@world.local",
			Callback: "invalid_callback_url",
		}))

		recorder := httptest.NewRecorder()
		request, err := http.NewRequest("POST", "http://localhost:8080/register", bodyReader)
		if err != nil {
			t.Fatal(err)
		}

		request.RemoteAddr = "127.0.0.1:51789"

		HttpAuthStartRegistrationProcess(testingCfg, db)(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Got %d, want %d", recorder.Code, http.StatusBadRequest)
		}

		if recorder.Body.String() != ErrInvalidCallbackUrl.Error() {
			t.Errorf("Got %s, want %s", recorder.Body.String(), ErrInvalidCallbackUrl.Error())
		}

		assertMailpitInboxIsEmpty(t, mailpit)
	})

	t.Run("returns 400 with ErrInvalidEmail when email is invalid", func(t *testing.T) {
		t.Parallel()
		mailpit := initializeMailpitAndDeleteAllMessages(t)
		defer func(mailpit *mailpitsuite.Api) {
			err := mailpit.Close()
			if err != nil {
				log.Println(err)
			}
		}(mailpit)

		db := openDatabase(t)
		bodyReader := bytes.NewReader(convertStructToJson(t, struct {
			Email    string `json:"email"`
			Callback string `json:"callback"`
		}{
			Email:    "invalid_email",
			Callback: "http://localhost:8080",
		}))

		recorder := httptest.NewRecorder()
		request, err := http.NewRequest("POST", "http://localhost:8080/register", bodyReader)
		if err != nil {
			t.Fatal(err)
		}

		request.RemoteAddr = "127.0.0.1:51789"

		HttpAuthStartRegistrationProcess(testingCfg, db)(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Got %d, want %d", recorder.Code, http.StatusBadRequest)
		}

		if recorder.Body.String() != ErrInvalidEmail.Error() {
			t.Errorf("Got %s, want %s", recorder.Body.String(), ErrInvalidEmail.Error())
		}

		assertMailpitInboxIsEmpty(t, mailpit)
	})

	t.Run("returns 400 with ErrUserWithGivenEmailAlreadyExists when user with given email already exists", func(t *testing.T) {
		t.Parallel()
		mailpit := initializeMailpitAndDeleteAllMessages(t)
		defer func(mailpit *mailpitsuite.Api) {
			err := mailpit.Close()
			if err != nil {
				log.Println(err)
			}
		}(mailpit)

		db := openDatabase(t)

		_, err := users.Create(db, "existing@user.local")
		if err != nil {
			t.Fatal(err)
		}

		bodyReader := bytes.NewReader(convertStructToJson(t, struct {
			Email    string `json:"email"`
			Callback string `json:"callback"`
		}{
			Email:    "existing@user.local",
			Callback: "http://localhost:8080",
		}))

		recorder := httptest.NewRecorder()
		request, err := http.NewRequest("POST", "http://localhost:8080/register", bodyReader)
		if err != nil {
			t.Fatal(err)
		}

		request.RemoteAddr = "127.0.0.1:51789"

		HttpAuthStartRegistrationProcess(testingCfg, db)(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Got %d, want %d", recorder.Code, http.StatusBadRequest)
		}

		if recorder.Body.String() != ErrUserWithGivenEmailAlreadyExists.Error() {
			t.Errorf("Got %s, want %s", recorder.Body.String(), ErrUserWithGivenEmailAlreadyExists.Error())
		}

		assertMailpitInboxIsEmpty(t, mailpit)
	})

	t.Run("returns 204 and sends registration email when everything is ok", func(t *testing.T) {
		t.Parallel()
		mailpit := initializeMailpitAndDeleteAllMessages(t)
		defer func(mailpit *mailpitsuite.Api) {
			err := mailpit.Close()
			if err != nil {
				log.Println(err)
			}
		}(mailpit)

		db := openDatabase(t)
		bodyReader := bytes.NewReader(convertStructToJson(t, struct {
			Email    string `json:"email"`
			Callback string `json:"callback"`
		}{
			Email:    "new@user.local",
			Callback: "http://localhost:8080",
		}))

		recorder := httptest.NewRecorder()
		request, err := http.NewRequest("POST", "http://localhost:8080/register", bodyReader)
		if err != nil {
			t.Fatal(err)
		}

		request.RemoteAddr = "127.0.0.1:51789"

		HttpAuthStartRegistrationProcess(testingCfg, db)(recorder, request)
		if recorder.Code != http.StatusNoContent {
			t.Errorf("Got %d, want %d", recorder.Code, http.StatusNoContent)
		}

		messages, err := mailpit.GetAllMessages()
		if err != nil {
			t.Fatalf("failed to get mailpit messages: %s", err.Error())
		}

		if len(messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(messages))
		}
	})
}
