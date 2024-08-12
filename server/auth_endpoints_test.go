package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
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
}

func convertStructToJson(t *testing.T, obj interface{}) []byte {
	result, err := json.Marshal(obj)

	if err != nil {
		t.Fatal(err)
	}

	return result
}

func TestHttpAuthLoginReturns400IfEmailIsInvalid(t *testing.T) {
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
}

func TestHttpAuthLoginReturns400IfCallbackIsInvalid(t *testing.T) {
	db := openDatabase(t)
	bodyReader := bytes.NewReader(convertStructToJson(t, struct {
		Email    string `json:"email"`
		Callback string `json:"callback"`
	}{
		Email:    "hello@world.local",
		Callback: "someinvalidurl'[]",
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
}
