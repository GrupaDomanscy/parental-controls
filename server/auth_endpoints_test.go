package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
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

	return db
}

func TestHttpAuthLoginReturns400IfJsonPayloadIsInvalid(t *testing.T) {
	mailpit, err := mailpitsuite.NewApi(mailpitExeFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer mailpit.Close()

	err = mailpit.DeleteAllMessages()
	if err != nil {
		t.Fatalf("failed to delete all mailpit messages: %s", err.Error())
	}

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

func TestHttpAuthLoginReturns400IfEmailIsInvalid(t *testing.T) {
	mailpit, err := mailpitsuite.NewApi(mailpitExeFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer mailpit.Close()

	err = mailpit.DeleteAllMessages()
	if err != nil {
		t.Fatalf("failed to delete all mailpit messages: %s", err.Error())
	}

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

	messages, err := mailpit.GetAllMessages()
	if err != nil {
		t.Fatalf("failed to get all mailpit messages: %s", err.Error())
	}

	if len(messages) != 0 {
		t.Errorf("Length of messages should be equal to 0, received %d", len(messages))
	}
}

func TestHttpAuthLoginReturns400IfCallbackIsInvalid(t *testing.T) {
	mailpit, err := mailpitsuite.NewApi(mailpitExeFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer mailpit.Close()

	err = mailpit.DeleteAllMessages()
	if err != nil {
		t.Fatalf("failed to delete all mailpit messages: %s", err.Error())
	}

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

	messages, err := mailpit.GetAllMessages()
	if err != nil {
		t.Fatalf("failed to get all mailpit messages: %s", err.Error())
	}

	if len(messages) != 0 {
		t.Errorf("Length of messages should be equal to 0, received %d", len(messages))
	}
}
