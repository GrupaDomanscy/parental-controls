package main

import (
	"bytes"
	"context"
	"database/sql"
	"domanscy.group/parental-controls/server/database"
	"domanscy.group/parental-controls/server/users"
	"domanscy.group/simplecache"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"mailpitsuite"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

var testingCfg = &ServerConfig{
	AppUrl:           "http://127.0.0.1:8080",
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
		t.Fatal(err)
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

func assertRegkeyStoreHasOneItemAndItMatchesTheRequestData(t *testing.T, regkeyStore *simplecache.Store, expectedEmail string, expectedCallback string) {
	keys := regkeyStore.GetAllKeys()

	if len(keys) != 1 {
		t.Errorf("Expected one key, received: %d", len(keys))
	}

	value, err := regkeyStore.Get(keys[0])
	if err != nil {
		t.Fatal(err)
	}

	if value != fmt.Sprintf("%s;%s", expectedEmail, expectedCallback) {
		t.Errorf("Expected %s;%s, received %s", expectedEmail, expectedCallback, value)
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
		defer func(mailpit *mailpitsuite.Api) {
			err := mailpit.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(mailpit)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		regkeyStore := simplecache.InitializeStore(ctx, time.Second)
		oneTimeAccessTokenStore := simplecache.InitializeStore(ctx, time.Minute)

		db := openDatabase(t)

		defer func(db *sql.DB) {
			err := db.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(db)

		bodyReader := strings.NewReader("notavalidjson")

		recorder := httptest.NewRecorder()
		request, err := http.NewRequest("GET", "http://localhost:8080/login", bodyReader)
		if err != nil {
			t.Fatal(err)
		}

		request.RemoteAddr = "127.0.0.1:51789"

		HttpAuthLogin(testingCfg, regkeyStore, oneTimeAccessTokenStore, db)(recorder, request)

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
		defer func(mailpit *mailpitsuite.Api) {
			err := mailpit.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(mailpit)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		regkeyStore := simplecache.InitializeStore(ctx, time.Second)
		oneTimeAccessTokenStore := simplecache.InitializeStore(ctx, time.Minute)

		db := openDatabase(t)
		defer func(db *sql.DB) {
			err := db.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(db)

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

		HttpAuthLogin(testingCfg, regkeyStore, oneTimeAccessTokenStore, db)(recorder, request)
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
		defer func(mailpit *mailpitsuite.Api) {
			err := mailpit.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(mailpit)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		regkeyStore := simplecache.InitializeStore(ctx, time.Second)
		oneTimeAccessTokenStore := simplecache.InitializeStore(ctx, time.Minute)
		db := openDatabase(t)
		defer func(db *sql.DB) {
			err := db.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(db)

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

		HttpAuthLogin(testingCfg, regkeyStore, oneTimeAccessTokenStore, db)(recorder, request)
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
		defer func(mailpit *mailpitsuite.Api) {
			err := mailpit.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(mailpit)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		regkeyStore := simplecache.InitializeStore(ctx, time.Second)
		oneTimeAccessTokenStore := simplecache.InitializeStore(ctx, time.Minute)

		db := openDatabase(t)
		defer func(db *sql.DB) {
			err := db.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(db)

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

		HttpAuthLogin(testingCfg, regkeyStore, oneTimeAccessTokenStore, db)(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Got %d, want %d", recorder.Code, http.StatusBadRequest)
		}

		if recorder.Body.String() != ErrUserWithGivenEmailDoesNotExist.Error() {
			t.Errorf("Got %s, want %s", recorder.Body.String(), ErrInvalidCallbackUrl.Error())
		}

		assertMailpitInboxIsEmpty(t, mailpit)
	})
}

//go:embed embeds_for_testing/auth_endpoints_test_embed_01.txt
var embed01 string

func TestHttpAuthStartRegistrationProcess(t *testing.T) {
	t.Parallel()

	t.Run("returns 400 with ErrInvalidCallbackUrl when callback is invalid", func(t *testing.T) {
		t.Parallel()
		mailpit := initializeMailpitAndDeleteAllMessages(t)
		defer func(mailpit *mailpitsuite.Api) {
			err := mailpit.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(mailpit)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		regkeyStore := simplecache.InitializeStore(ctx, time.Second)
		oneTimeAccessTokenStore := simplecache.InitializeStore(ctx, time.Minute)

		db := openDatabase(t)
		defer func(db *sql.DB) {
			err := db.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(db)

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

		HttpAuthStartRegistrationProcess(testingCfg, regkeyStore, oneTimeAccessTokenStore, db)(recorder, request)
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
				t.Fatal(err)
			}
		}(mailpit)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		regkeyStore := simplecache.InitializeStore(ctx, time.Second)
		oneTimeAccessTokenStore := simplecache.InitializeStore(ctx, time.Minute)

		db := openDatabase(t)
		defer func(db *sql.DB) {
			err := db.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(db)

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

		HttpAuthStartRegistrationProcess(testingCfg, regkeyStore, oneTimeAccessTokenStore, db)(recorder, request)
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
				t.Fatal(err)
			}
		}(mailpit)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		regkeyStore := simplecache.InitializeStore(ctx, time.Second)
		oneTimeAccessTokenStore := simplecache.InitializeStore(ctx, time.Minute)

		db := openDatabase(t)
		defer func(db *sql.DB) {
			err := db.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(db)

		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}

		_, err = users.Create(tx, "existing@user.local")
		if err != nil {
			txErr := tx.Rollback()
			if txErr != nil {
				t.Error(txErr)
			}

			t.Fatal(err)
		}

		err = tx.Commit()
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

		requestCtx, cancelRequestCtx := context.WithCancel(request.Context())
		request = request.WithContext(requestCtx)

		request.RemoteAddr = "127.0.0.1:51789"

		HttpAuthStartRegistrationProcess(testingCfg, regkeyStore, oneTimeAccessTokenStore, db)(recorder, request)
		cancelRequestCtx()

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Got %d, want %d", recorder.Code, http.StatusBadRequest)
		}

		if recorder.Body.String() != ErrUserWithGivenEmailAlreadyExists.Error() {
			t.Errorf("Got %s, want %s", recorder.Body.String(), ErrUserWithGivenEmailAlreadyExists.Error())
		}

		assertMailpitInboxIsEmpty(t, mailpit)
	})

	t.Run("returns 204, sends registration email without non-official site warning and puts correct value in regkey cache when everything is ok", func(t *testing.T) {
		t.Parallel()
		mailpit := initializeMailpitAndDeleteAllMessages(t)
		defer func(mailpit *mailpitsuite.Api) {
			err := mailpit.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(mailpit)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		regkeyStore := simplecache.InitializeStore(ctx, time.Second)
		oneTimeAccessTokenStore := simplecache.InitializeStore(ctx, time.Minute)

		db := openDatabase(t)
		defer func(db *sql.DB) {
			err := db.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(db)

		reqBody := struct {
			Email    string `json:"email"`
			Callback string `json:"callback"`
		}{
			Email:    "new@user.local",
			Callback: "http://localhost:8080",
		}

		bodyReader := bytes.NewReader(convertStructToJson(t, reqBody))

		recorder := httptest.NewRecorder()
		request, err := http.NewRequest("POST", "http://localhost:8080/register", bodyReader)
		if err != nil {
			t.Fatal(err)
		}

		request.RemoteAddr = "127.0.0.1:51789"

		HttpAuthStartRegistrationProcess(testingCfg, regkeyStore, oneTimeAccessTokenStore, db)(recorder, request)
		if recorder.Code != http.StatusNoContent {
			t.Errorf("Got %d, want %d", recorder.Code, http.StatusNoContent)
		}

		messages, err := mailpit.GetAllMessages()
		if err != nil {
			t.Fatalf("failed to get mailpit messages: %s", err.Error())
		}

		if len(messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(messages))
			t.FailNow()
		}

		messageSummary, err := mailpit.GetMessageSummary(messages[0].ID)
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(messageSummary.HTML, embed01) {
			t.Errorf("Email is invalid.")
		}

		assertRegkeyStoreHasOneItemAndItMatchesTheRequestData(t, regkeyStore, reqBody.Email, reqBody.Callback)
	})

	t.Run("returns 204 and sends registration email with warning about non-official site when everything is ok and callback is not from official site", func(t *testing.T) {
		t.Parallel()
		mailpit := initializeMailpitAndDeleteAllMessages(t)
		defer func(mailpit *mailpitsuite.Api) {
			err := mailpit.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(mailpit)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		regkeyStore := simplecache.InitializeStore(ctx, time.Second)
		oneTimeAccessTokenStore := simplecache.InitializeStore(ctx, time.Minute)

		db := openDatabase(t)
		defer func(db *sql.DB) {
			err := db.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(db)

		reqBody := struct {
			Email    string `json:"email"`
			Callback string `json:"callback"`
		}{
			Email:    "new@user.local",
			Callback: "http://officialinstance.local/callback",
		}

		bodyReader := bytes.NewReader(convertStructToJson(t, reqBody))

		recorder := httptest.NewRecorder()
		request, err := http.NewRequest("POST", "http://localhost:8080/register", bodyReader)
		if err != nil {
			t.Fatal(err)
		}

		request.RemoteAddr = "127.0.0.1:51789"

		HttpAuthStartRegistrationProcess(testingCfg, regkeyStore, oneTimeAccessTokenStore, db)(recorder, request)
		if recorder.Code != http.StatusNoContent {
			t.Errorf("Got %d, want %d", recorder.Code, http.StatusNoContent)
		}

		messages, err := mailpit.GetAllMessages()
		if err != nil {
			t.Fatalf("failed to get mailpit messages: %s", err.Error())
		}

		if len(messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(messages))
		} else {
			messageSummary, err := mailpit.GetMessageSummary(messages[0].ID)
			if err != nil {
				t.Fatal(err)
			}

			if strings.Contains(messageSummary.HTML, embed01) {
				t.Errorf("Email is invalid.")
			}

			assertRegkeyStoreHasOneItemAndItMatchesTheRequestData(t, regkeyStore, reqBody.Email, reqBody.Callback)
		}
	})
}

func TestHttpAuthFinishRegistrationProcess(t *testing.T) {
	t.Parallel()

	t.Run("generates one time access token, redirects to callback url with generated access token and creates the user when everything is ok", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		regkeyStore := simplecache.InitializeStore(ctx, time.Minute)
		oneTimeAccessTokenStore := simplecache.InitializeStore(ctx, time.Minute)

		value, err := regkeyStore.PutAndGenerateRandomKeyForValue("new@user.local;http://officialinstance.local/callback")
		if err != nil {
			t.Fatal(err)
		}

		db := openDatabase(t)
		defer func(db *sql.DB) {
			err := db.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(db)

		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "http://localhost:8080/finish_registration/"+url.PathEscape(value), nil)
		if err != nil {
			t.Fatal(err)
		}

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("regkey", url.PathEscape(value))

		request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, rctx))

		HttpAuthFinishRegistrationProcess(testingCfg, regkeyStore, oneTimeAccessTokenStore, db)(recorder, request)

		if recorder.Code != http.StatusTemporaryRedirect {
			t.Errorf("Got %d, want %d, response body: %s", recorder.Code, http.StatusTemporaryRedirect, recorder.Body.String())
		}

		keys := oneTimeAccessTokenStore.GetAllKeys()
		if len(keys) != 1 {
			t.Errorf("Expected 1 key in store, received: %d", len(keys))
		}

		location := recorder.Header().Get("Location")
		expectedUrl, err := url.Parse("http://officialinstance.local/callback")
		if err != nil {
			t.Fatal(err)
		}

		queryParams := expectedUrl.Query()
		queryParams.Set("oneTimeAccessToken", keys[0])

		expectedUrl.RawQuery = queryParams.Encode()

		if location != expectedUrl.String() {
			t.Errorf("Expected location to be %s, received %s", expectedUrl.String(), location)
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

		user, err := users.FindOneByEmail(tx, "new@user.local")
		if err != nil {
			t.Fatal(err)
		}

		payload, err := oneTimeAccessTokenStore.Get(keys[0])
		if err != nil {
			t.Fatal(err)
		}

		if payload != fmt.Sprintf("userId:%d", user.Id) {
			t.Errorf("Expected payload to be: %s, received %s", fmt.Sprintf("userId:%d", user.Id), payload)
		}
	})

	t.Run("disables already used regkey and returns 400 if user tries to use it", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		regkeyStore := simplecache.InitializeStore(ctx, time.Minute)
		oneTimeAccessTokenStore := simplecache.InitializeStore(ctx, time.Minute)

		value, err := regkeyStore.PutAndGenerateRandomKeyForValue("new@user.local;http://officialinstance.local/callback")
		if err != nil {
			t.Fatal(err)
		}

		db := openDatabase(t)
		defer func(db *sql.DB) {
			err := db.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(db)

		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "http://localhost:8080/finish_registration/"+url.PathEscape(value), nil)
		if err != nil {
			t.Fatal(err)
		}

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("regkey", url.PathEscape(value))

		request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, rctx))

		HttpAuthFinishRegistrationProcess(testingCfg, regkeyStore, oneTimeAccessTokenStore, db)(recorder, request)
		if recorder.Code != http.StatusTemporaryRedirect {
			t.Errorf("Got %d, want %d, response body: %s", recorder.Code, http.StatusTemporaryRedirect, recorder.Body.String())
		}

		recorder = httptest.NewRecorder()

		HttpAuthFinishRegistrationProcess(testingCfg, regkeyStore, oneTimeAccessTokenStore, db)(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Got %d, want %d, response body: %s", recorder.Code, http.StatusTemporaryRedirect, recorder.Body.String())
		}

		responseBody := recorder.Body.String()

		if responseBody != ErrInvalidRegistrationKey.Error() {
			t.Errorf("Expected %s, received %s", ErrInvalidRegistrationKey.Error(), responseBody)
		}
	})
}
