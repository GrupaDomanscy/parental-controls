package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"mailpitsuite"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"domanscy.group/parental-controls/server/database"
	"domanscy.group/parental-controls/server/users"
	"domanscy.group/rckstrvcache"
	"github.com/go-chi/chi"
)

var testingCfg = &ServerConfig{
	AppUrl:        "http://127.0.0.1:8080",
	ServerAddress: "127.0.0.1",
	ServerPort:    8080,

	EmailFromAddress: "test@parental-controls.local",
	SmtpAddress:      "127.0.0.1",
	SmtpPort:         1025,

	BearerTokenPrivateKey: rsaMustGenerateKey(),

	DatabaseUrl: ":memory:",
}

func rsaMustGenerateKey() *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatal(err)
	}

	return key
}

func doTFatalIfErr(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
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

func assertRegkeyStoreHasOneItemAndItMatchesTheRequestData(t *testing.T, regkeyStore *rckstrvcache.Store, expectedEmail string, expectedCallback string) {
	keys, err := regkeyStore.GetAllKeys()
	if err != nil {
		t.Fatal(err)
	}

	if len(keys) != 1 {
		t.Errorf("Expected one key, received: %d", len(keys))
	}

	value, exists, err := regkeyStore.Get(keys[0])
	if err != nil {
		t.Fatal(err)
	}

	if !exists {
		t.Errorf("expected regkey to exist")
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
	mailpit, err := mailpitsuite.NewApi(getMailpitExecutableFilePath())
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

		regkeyStore, regkeyErrCh, err := rckstrvcache.InitializeStore(time.Second)
		oneTimeAccessTokenStore, otatErrCh, err := rckstrvcache.InitializeStore(time.Minute)

		defer func(regkeyStore *rckstrvcache.Store) {
			err := regkeyStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(regkeyStore)
		defer func(oneTimeAccessTokenStore *rckstrvcache.Store) {
			err := oneTimeAccessTokenStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(oneTimeAccessTokenStore)

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

		select {
		case err = <-regkeyErrCh:
			log.Fatal(err)
		case err = <-otatErrCh:
			log.Fatal(err)
		default:
		}
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

		regkeyStore, regkeyErrCh, err := rckstrvcache.InitializeStore(time.Second)
		oneTimeAccessTokenStore, otatErrCh, err := rckstrvcache.InitializeStore(time.Minute)

		defer func(regkeyStore *rckstrvcache.Store) {
			err := regkeyStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(regkeyStore)
		defer func(oneTimeAccessTokenStore *rckstrvcache.Store) {
			err := oneTimeAccessTokenStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(oneTimeAccessTokenStore)

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

		select {
		case err = <-regkeyErrCh:
			log.Fatal(err)
		case err = <-otatErrCh:
			log.Fatal(err)
		default:
		}
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

		regkeyStore, regkeyErrCh, err := rckstrvcache.InitializeStore(time.Second)
		oneTimeAccessTokenStore, otatErrCh, err := rckstrvcache.InitializeStore(time.Minute)

		defer func(regkeyStore *rckstrvcache.Store) {
			err := regkeyStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(regkeyStore)
		defer func(oneTimeAccessTokenStore *rckstrvcache.Store) {
			err := oneTimeAccessTokenStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(oneTimeAccessTokenStore)

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

		select {
		case err = <-regkeyErrCh:
			log.Fatal(err)
		case err = <-otatErrCh:
			log.Fatal(err)
		default:
		}
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

		regkeyStore, regkeyErrCh, err := rckstrvcache.InitializeStore(time.Second)
		oneTimeAccessTokenStore, otatErrCh, err := rckstrvcache.InitializeStore(time.Minute)

		defer func(regkeyStore *rckstrvcache.Store) {
			err := regkeyStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(regkeyStore)
		defer func(oneTimeAccessTokenStore *rckstrvcache.Store) {
			err := oneTimeAccessTokenStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(oneTimeAccessTokenStore)

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

		select {
		case err = <-regkeyErrCh:
			log.Fatal(err)
		case err = <-otatErrCh:
			log.Fatal(err)
		default:
		}
	})
}

//go:embed embeds_for_testing/auth_endpoints_test_embed_01.txt
var embed01 string
var embed01Template *template.Template = template.Must(template.New("embed01").Parse(embed01))

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

		regkeyStore, regkeyErrCh, err := rckstrvcache.InitializeStore(time.Second)
		oneTimeAccessTokenStore, otatErrCh, err := rckstrvcache.InitializeStore(time.Minute)

		defer func(regkeyStore *rckstrvcache.Store) {
			err := regkeyStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(regkeyStore)
		defer func(oneTimeAccessTokenStore *rckstrvcache.Store) {
			err := oneTimeAccessTokenStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(oneTimeAccessTokenStore)

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

		select {
		case err = <-regkeyErrCh:
			log.Fatal(err)
		case err = <-otatErrCh:
			log.Fatal(err)
		default:
		}
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

		regkeyStore, regkeyErrCh, err := rckstrvcache.InitializeStore(time.Second)
		oneTimeAccessTokenStore, otatErrCh, err := rckstrvcache.InitializeStore(time.Minute)

		defer func(regkeyStore *rckstrvcache.Store) {
			err := regkeyStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(regkeyStore)
		defer func(oneTimeAccessTokenStore *rckstrvcache.Store) {
			err := oneTimeAccessTokenStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(oneTimeAccessTokenStore)

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

		select {
		case err = <-regkeyErrCh:
			log.Fatal(err)
		case err = <-otatErrCh:
			log.Fatal(err)
		default:
		}
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

		regkeyStore, regkeyErrCh, err := rckstrvcache.InitializeStore(time.Second)
		oneTimeAccessTokenStore, otatErrCh, err := rckstrvcache.InitializeStore(time.Minute)

		defer func(regkeyStore *rckstrvcache.Store) {
			err := regkeyStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(regkeyStore)
		defer func(oneTimeAccessTokenStore *rckstrvcache.Store) {
			err := oneTimeAccessTokenStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(oneTimeAccessTokenStore)

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

		select {
		case err = <-regkeyErrCh:
			log.Fatal(err)
		case err = <-otatErrCh:
			log.Fatal(err)
		default:
		}
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

		regkeyStore, regkeyErrCh, err := rckstrvcache.InitializeStore(time.Second)
		oneTimeAccessTokenStore, otatErrCh, err := rckstrvcache.InitializeStore(time.Minute)

		defer func(regkeyStore *rckstrvcache.Store) {
			err := regkeyStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(regkeyStore)
		defer func(oneTimeAccessTokenStore *rckstrvcache.Store) {
			err := oneTimeAccessTokenStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(oneTimeAccessTokenStore)

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

		assertRegkeyStoreHasOneItemAndItMatchesTheRequestData(t, regkeyStore, reqBody.Email, reqBody.Callback)

		buffer := bytes.NewBuffer([]byte{})

		allRegkeys, err := regkeyStore.GetAllKeys()
		if err != nil {
			t.Fatal(err)
		}

		if len(allRegkeys) != 1 {
			t.Fatalf("Expected one regkey, received: %d", len(allRegkeys))
		}

		regkey := allRegkeys[0]

		err = embed01Template.ExecuteTemplate(buffer, "embed01", struct {
			AppUrl string
			Token  string
		}{
			AppUrl: testingCfg.AppUrl,
			Token:  regkey,
		})
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(messageSummary.HTML, buffer.String()) {
			t.Errorf("Email is invalid.\n\n Expected:\n%s\n\nReceived:\n%s", buffer.String(), messageSummary.HTML)
		}

		select {
		case err = <-regkeyErrCh:
			log.Fatal(err)
		case err = <-otatErrCh:
			log.Fatal(err)
		default:
		}
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

		regkeyStore, regkeyErrCh, err := rckstrvcache.InitializeStore(time.Second)
		oneTimeAccessTokenStore, otatErrCh, err := rckstrvcache.InitializeStore(time.Minute)

		defer func(regkeyStore *rckstrvcache.Store) {
			err := regkeyStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(regkeyStore)
		defer func(oneTimeAccessTokenStore *rckstrvcache.Store) {
			err := oneTimeAccessTokenStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(oneTimeAccessTokenStore)

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

		select {
		case err = <-regkeyErrCh:
			log.Fatal(err)
		case err = <-otatErrCh:
			log.Fatal(err)
		default:
		}
	})
}

func TestHttpAuthFinishRegistrationProcess(t *testing.T) {
	t.Parallel()

	t.Run("generates one time access token, redirects to callback url with generated access token and creates the user when everything is ok", func(t *testing.T) {
		t.Parallel()

		regkeyStore, regkeyErrCh, err := rckstrvcache.InitializeStore(time.Second)
		oneTimeAccessTokenStore, otatErrCh, err := rckstrvcache.InitializeStore(time.Minute)

		defer func(regkeyStore *rckstrvcache.Store) {
			err := regkeyStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(regkeyStore)
		defer func(oneTimeAccessTokenStore *rckstrvcache.Store) {
			err := oneTimeAccessTokenStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(oneTimeAccessTokenStore)

		value, err := regkeyStore.Put("new@user.local;http://officialinstance.local/callback")
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

		keys, err := oneTimeAccessTokenStore.GetAllKeys()
		if err != nil {
			t.Fatal(err)
		}

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

		user, err := users.FindOneByEmail(tx, "new@user.local")
		if err != nil {
			txErr := tx.Rollback()
			if txErr != nil {
				err = errors.Join(err, txErr)
			}

			t.Fatal(err)
		}

		err = tx.Commit()
		if err != nil {
			t.Fatal(err)
		}

		payload, exists, err := oneTimeAccessTokenStore.Get(keys[0])
		if err != nil {
			t.Fatal(err)
		}

		if !exists {
			t.Errorf("expected one time access token to exist in cache")
		}

		if payload != fmt.Sprintf("userId:%d", user.Id) {
			t.Errorf("Expected payload to be: %s, received %s", fmt.Sprintf("userId:%d", user.Id), payload)
		}

		select {
		case err = <-regkeyErrCh:
			log.Fatal(err)
		case err = <-otatErrCh:
			log.Fatal(err)
		default:
		}
	})

	t.Run("disables already used regkey and returns 400 if user tries to use it", func(t *testing.T) {
		t.Parallel()

		regkeyStore, regkeyErrCh, err := rckstrvcache.InitializeStore(time.Second)
		oneTimeAccessTokenStore, otatErrCh, err := rckstrvcache.InitializeStore(time.Minute)

		defer func(regkeyStore *rckstrvcache.Store) {
			err := regkeyStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(regkeyStore)
		defer func(oneTimeAccessTokenStore *rckstrvcache.Store) {
			err := oneTimeAccessTokenStore.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(oneTimeAccessTokenStore)

		value, err := regkeyStore.Put("new@user.local;http://officialinstance.local/callback")
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

		select {
		case err = <-regkeyErrCh:
			log.Fatal(err)
		case err = <-otatErrCh:
			log.Fatal(err)
		default:
		}
	})
}

func TestHttpAuthGetBearerTokenFromOtat(t *testing.T) {
	t.Run("returns 400 if provided otat does not exist", func(t *testing.T) {
		regkeyStore, regkeyErrCh, err := rckstrvcache.InitializeStore(time.Second)
		oneTimeAccessTokenStore, otatErrCh, err := rckstrvcache.InitializeStore(time.Minute)

		defer func(regkeyStore *rckstrvcache.Store, t *testing.T) {
			doTFatalIfErr(t, regkeyStore.Close())
		}(regkeyStore, t)
		defer func(oneTimeAccessTokenStore *rckstrvcache.Store, t *testing.T) {
			doTFatalIfErr(t, oneTimeAccessTokenStore.Close())
		}(oneTimeAccessTokenStore, t)

		db := openDatabase(t)
		defer func(db *sql.DB, t *testing.T) {
			doTFatalIfErr(t, db.Close())
		}(db, t)

		recorder := httptest.NewRecorder()

		request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/get_bearer_token_from_otat/somenonexistentotat", testingCfg.AppUrl), nil)
		if err != nil {
			t.Fatal(err)
		}

		HttpAuthGetBearerTokenFromOtat(testingCfg, regkeyStore, oneTimeAccessTokenStore, db)(recorder, request)
		if recorder.Result().StatusCode != 400 {
			t.Errorf("expected status code 400, received %d", recorder.Result().StatusCode)
		}

		if recorder.Body.String() != ErrInvalidOtat.Error() {
			t.Errorf("expected body \"%s\", received \"%s\"", ErrInvalidOtat.Error(), recorder.Body.String())
		}

		select {
		case err = <-regkeyErrCh:
			log.Fatal(err)
		case err = <-otatErrCh:
			log.Fatal(err)
		default:
		}
	})

	t.Run("returns 400 if otat is expired", func(t *testing.T) {
		regkeyStore, regkeyErrCh, err := rckstrvcache.InitializeStore(time.Second)
		oneTimeAccessTokenStore, otatErrCh, err := rckstrvcache.InitializeStore(time.Second)

		defer func(regkeyStore *rckstrvcache.Store, t *testing.T) {
			doTFatalIfErr(t, regkeyStore.Close())
		}(regkeyStore, t)
		defer func(oneTimeAccessTokenStore *rckstrvcache.Store, t *testing.T) {
			doTFatalIfErr(t, oneTimeAccessTokenStore.Close())
		}(oneTimeAccessTokenStore, t)

		db := openDatabase(t)
		defer func(db *sql.DB, t *testing.T) {
			doTFatalIfErr(t, db.Close())
		}(db, t)

		key, err := oneTimeAccessTokenStore.Put("userId:2")
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Second * 2)

		recorder := httptest.NewRecorder()

		key = url.PathEscape(key)

		chiRouteCtx := chi.NewRouteContext()
		chiRouteCtx.URLParams.Add("otat", key)

		request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/get_bearer_token_from_otat/%s", testingCfg.AppUrl, key), nil)
		if err != nil {
			t.Fatal(err)
		}

		request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, chiRouteCtx))

		HttpAuthGetBearerTokenFromOtat(testingCfg, regkeyStore, oneTimeAccessTokenStore, db)(recorder, request)
		if recorder.Result().StatusCode != 400 {
			t.Errorf("expected status code 400, received %d", recorder.Result().StatusCode)
		}

		if recorder.Body.String() != ErrInvalidOtat.Error() {
			t.Errorf("expected body \"%s\", received \"%s\"", ErrInvalidOtat.Error(), recorder.Body.String())
		}

		select {
		case err = <-regkeyErrCh:
			log.Fatal(err)
		case err = <-otatErrCh:
			log.Fatal(err)
		default:
		}
	})

	t.Run("returns 400 if owner of the otat (user) has been removed", func(t *testing.T) {
		regkeyStore, regkeyErrCh, err := rckstrvcache.InitializeStore(time.Second)
		oneTimeAccessTokenStore, otatErrCh, err := rckstrvcache.InitializeStore(time.Minute)

		defer func(regkeyStore *rckstrvcache.Store, t *testing.T) {
			doTFatalIfErr(t, regkeyStore.Close())
		}(regkeyStore, t)
		defer func(oneTimeAccessTokenStore *rckstrvcache.Store, t *testing.T) {
			doTFatalIfErr(t, oneTimeAccessTokenStore.Close())
		}(oneTimeAccessTokenStore, t)

		db := openDatabase(t)
		defer func(db *sql.DB, t *testing.T) {
			doTFatalIfErr(t, db.Close())
		}(db, t)

		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}

		userId, err := users.Create(tx, "user@localhost.local")
		if err != nil {
			t.Fatal(err)
		}

		key, err := oneTimeAccessTokenStore.Put(fmt.Sprintf("userId:%d", userId))
		if err != nil {
			t.Fatal(err)
		}

		key = url.PathEscape(key)

		// remove the user
		err = tx.Rollback()
		if err != nil {
			t.Fatal(err)
		}

		recorder := httptest.NewRecorder()

		chiRouteCtx := chi.NewRouteContext()
		chiRouteCtx.URLParams.Add("otat", key)

		request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/get_bearer_token_from_otat/%s", testingCfg.AppUrl, key), nil)
		if err != nil {
			t.Fatal(err)
		}

		request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, chiRouteCtx))

		HttpAuthGetBearerTokenFromOtat(testingCfg, regkeyStore, oneTimeAccessTokenStore, db)(recorder, request)
		if recorder.Result().StatusCode != 400 {
			t.Errorf("expected status code 400, received %d", recorder.Result().StatusCode)
		}

		if recorder.Body.String() != ErrInvalidOtat.Error() {
			t.Errorf("expected body \"%s\", received \"%s\"", ErrInvalidOtat.Error(), recorder.Body.String())
		}

		select {
		case err = <-regkeyErrCh:
			log.Fatal(err)
		case err = <-otatErrCh:
			log.Fatal(err)
		default:
		}
	})

	t.Run("generates valid bearer token and returns 200 ok", func(t *testing.T) {
		regkeyStore, regkeyErrCh, err := rckstrvcache.InitializeStore(time.Second)
		oneTimeAccessTokenStore, otatErrCh, err := rckstrvcache.InitializeStore(time.Minute)

		defer func(regkeyStore *rckstrvcache.Store, t *testing.T) {
			doTFatalIfErr(t, regkeyStore.Close())
		}(regkeyStore, t)
		defer func(oneTimeAccessTokenStore *rckstrvcache.Store, t *testing.T) {
			doTFatalIfErr(t, oneTimeAccessTokenStore.Close())
		}(oneTimeAccessTokenStore, t)

		db := openDatabase(t)
		defer func(db *sql.DB, t *testing.T) {
			doTFatalIfErr(t, db.Close())
		}(db, t)

		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}

		userId, err := users.Create(tx, "user@localhost.local")
		if err != nil {
			t.Fatal(err)
		}

		key, err := oneTimeAccessTokenStore.Put(fmt.Sprintf("userId:%d", userId))
		if err != nil {
			t.Fatal(err)
		}

		key = url.PathEscape(key)

		err = tx.Commit()
		if err != nil {
			t.Fatal(err)
		}

		recorder := httptest.NewRecorder()

		chiRouteCtx := chi.NewRouteContext()
		chiRouteCtx.URLParams.Add("otat", key)

		request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/get_bearer_token_from_otat/%s", testingCfg.AppUrl, key), nil)
		if err != nil {
			t.Fatal(err)
		}

		request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, chiRouteCtx))

		HttpAuthGetBearerTokenFromOtat(testingCfg, regkeyStore, oneTimeAccessTokenStore, db)(recorder, request)
		if recorder.Result().StatusCode != 200 {
			t.Errorf("expected status code 200, received %d", recorder.Result().StatusCode)
		}

		token := recorder.Body.String()

		userIdFromToken, err := GetUserIdFromBearerToken(testingCfg.BearerTokenPrivateKey, []byte(token))
		if err != nil {
			t.Fatal(err)
		}

		if userIdFromToken != userId {
			t.Errorf("user id from bearer token is invalid. expected %d, received %d", userId, userIdFromToken)
		}

		select {
		case err = <-regkeyErrCh:
			log.Fatal(err)
		case err = <-otatErrCh:
			log.Fatal(err)
		default:
		}
	})
}
