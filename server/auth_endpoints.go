package main

import (
	"bytes"
	"database/sql"
	"domanscy.group/parental-controls/server/users"
	"domanscy.group/simplecache"
	_ "embed"
	"errors"
	"fmt"
	"github.com/go-chi/chi"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
)

var ErrUserWithGivenEmailDoesNotExist = errors.New("user with given email does not exist")

func HttpAuthLogin(cfg *ServerConfig, _ *simplecache.Store, _ *simplecache.Store, db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		type RequestBody struct {
			Email    string `json:"email"`
			Callback string `json:"callback"`
		}

		var requestBody RequestBody

		if err := decodeJsonRequestBodyAndSendHttpErrorIfInvalid(w, r, &requestBody); err != nil {
			return
		}

		requestBody.Email = strings.ToLower(requestBody.Email)

		if err := parseEmailAddressAndHandleErrorIfInvalid(w, r, requestBody.Email); err != nil {
			return
		}

		if _, err := parseUrlAndHandleErrorIfInvalid(w, r, requestBody.Callback); err != nil {
			return
		}

		tx, err := db.BeginTx(r.Context(), nil)
		if err != nil {
			respondWith500(w, r, "")
			log.Printf("error occured while trying to start a transaction: %v", err)
			return
		}

		defer func(tx *sql.Tx) {
			err := tx.Commit()
			if err != nil {
				log.Printf("failed to commit the transaction: %v", err)
			}
		}(tx)

		user, err := users.FindOneByEmail(tx, requestBody.Email)
		if err != nil {
			txErr := tx.Rollback()
			if txErr != nil {
				log.Printf("failed to rollback the transaction: %v", txErr)
			}

			respondWith500(w, r, "")
			log.Printf("error occured while trying to find user by email: %v", err)
			return
		}

		if user == nil {
			txErr := tx.Rollback()
			if txErr != nil {
				log.Printf("failed to rollback the transaction: %v", txErr)
			}

			respondWith400(w, r, ErrUserWithGivenEmailDoesNotExist.Error())
			return
		}

		var message strings.Builder

		message.WriteString(fmt.Sprintf("From: %s\r\n", cfg.EmailFromAddress))
		message.WriteString(fmt.Sprintf("To: %s\r\n", requestBody.Email))
		message.WriteString(fmt.Sprintf("Subject: %s\r\n", "Zaloguj się"))
		message.WriteString(fmt.Sprintf("\r\n"))
		message.WriteString(fmt.Sprintf("Hello world!\r\n"))

		var mailBody strings.Builder

		mailBody.WriteString("Otrzymaliśmy prośbę o zalogowanie się do systemu.<br />")
		mailBody.WriteString(fmt.Sprintf("Prosimy o potwierdzenie czy ta osoba może się zalogować.<br />"))
		mailBody.WriteString(fmt.Sprintf("<br />"))

		ip, err := getIPAddressFromRequest(w, r)
		if err != nil {
			txErr := tx.Rollback()
			if txErr != nil {
				log.Printf("failed to rollback the transaction: %v", txErr)
			}

			log.Println(err)
			respondWith400(w, r, err.Error())
			return
		}

		mailBody.WriteString(fmt.Sprintf("Adres IP logowania: %s<br />", ip))

		mailBody.WriteString(fmt.Sprintf("<br />"))

		mailBody.WriteString(fmt.Sprintf("<a href=\"#\">Odrzuć</a> "))
		mailBody.WriteString(fmt.Sprintf("<a href=\"#\">Zezwól</a>"))

		err = sendMailAndHandleError(
			w, r,
			cfg.SmtpAddress,
			cfg.SmtpPort,
			cfg.EmailFromAddress,
			requestBody.Email,
			"Potwierdź logowanie do kontroli rodzicielskiej",
			mailBody.String(),
		)
		if err != nil {
			txErr := tx.Rollback()
			if txErr != nil {
				log.Printf("failed to rollback the transaction: %v", txErr)
			}

			respondWith500(w, r, "")
			return
		}

		w.WriteHeader(204)
	}
}

//go:embed mail_templates/register.gohtml
var startRegistrationProcessEmailBody string
var startRegistrationProcessEmailTemplate = template.Must(template.New("email_template").Parse(startRegistrationProcessEmailBody))

var ErrUserWithGivenEmailAlreadyExists = errors.New("user with given email already exists")

func HttpAuthStartRegistrationProcess(cfg *ServerConfig, regkeysStore *simplecache.Store, _ *simplecache.Store, db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		type RequestBody struct {
			Email    string `json:"email"`
			Callback string `json:"callback"`
		}

		var requestBody RequestBody

		if err := decodeJsonRequestBodyAndSendHttpErrorIfInvalid(w, r, &requestBody); err != nil {
			return
		}

		if err := parseEmailAddressAndHandleErrorIfInvalid(w, r, requestBody.Email); err != nil {
			return
		}

		callbackUrl, err := parseUrlAndHandleErrorIfInvalid(w, r, requestBody.Callback)
		if err != nil {
			return
		}

		tx, err := db.BeginTx(r.Context(), nil)
		if err != nil {
			respondWith500(w, r, "")
			log.Printf("error occured while trying to start a transaction: %v", err)
			return
		}

		defer func(tx *sql.Tx) {
			err := tx.Commit()
			if err != nil {
				log.Printf("failed to commit the transaction: %v", err)
			}
		}(tx)

		user, err := users.FindOneByEmail(tx, requestBody.Email)
		if err != nil {
			txErr := tx.Rollback()
			if txErr != nil {
				log.Printf("failed to rollback the transaction: %v", txErr)
			}

			respondWith500(w, r, "")
			log.Printf("error occured while trying to find user by email: %v", err)
			return
		}

		if user != nil {
			txErr := tx.Rollback()
			if txErr != nil {
				log.Printf("failed to rollback the transaction: %v", txErr)
			}

			respondWith400(w, r, ErrUserWithGivenEmailAlreadyExists.Error())
			return
		}

		regkey, err := regkeysStore.PutAndGenerateRandomKeyForValue(requestBody.Email + ";" + callbackUrl.String())
		if err != nil {
			txErr := tx.Rollback()
			if txErr != nil {
				log.Printf("failed to rollback the transaction: %v", txErr)
			}

			respondWith500(w, r, "")
			log.Printf("an error occured while trying to generate new regkey for email '%s': %v", requestBody.Email, err)
			return
		}

		emailBody := bytes.NewBuffer([]byte{})
		err = startRegistrationProcessEmailTemplate.ExecuteTemplate(emailBody, "email_template", struct {
			InstanceAddr       string
			IsOfficialInstance bool
			Link               string
		}{
			InstanceAddr:       callbackUrl.Host,
			IsOfficialInstance: callbackUrl.Host == "officialinstance.local", //TODO
			Link:               fmt.Sprintf("%s/finish_registration/%s", cfg.AppUrl, url.PathEscape(regkey)),
		})
		if err != nil {
			regkeysStore.Delete(regkey)

			txErr := tx.Rollback()
			if txErr != nil {
				log.Printf("failed to rollback the transaction: %v", txErr)
			}

			respondWith500(w, r, "")
			log.Printf("an error occured while trying to generate email body: %v", err)
			return
		}

		err = sendMailAndHandleError(
			w, r,
			cfg.SmtpAddress,
			cfg.SmtpPort,
			cfg.EmailFromAddress,
			requestBody.Email,
			"Potwierdź rejestracje w kontroli rodzicielskiej",
			emailBody.String(),
		)
		if err != nil {
			regkeysStore.Delete(regkey)

			txErr := tx.Rollback()
			if txErr != nil {
				log.Printf("failed to rollback the transaction: %v", txErr)
			}

			log.Printf("an error occured while trying to send mail: %v", err)
			respondWith500(w, r, "")
			return
		}

		w.WriteHeader(204)
	}
}

var ErrRegistrationKeyCannotBeEmpty = errors.New("registration key can not be empty")
var ErrInvalidRegistrationKey = errors.New("invalid registration key")

func HttpAuthFinishRegistrationProcess(_ *ServerConfig, regkeyStore *simplecache.Store, oneTimeAccessTokenStore *simplecache.Store, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		regkey := chi.URLParam(r, "regkey")
		if regkey == "" {
			respondWith400(w, r, ErrRegistrationKeyCannotBeEmpty.Error())
			return
		}

		regkey, err := url.PathUnescape(regkey)
		if err != nil {
			respondWith400(w, r, ErrInvalidRegistrationKey.Error())
			return
		}

		cachePayload, err := regkeyStore.Get(regkey)
		if err != nil {
			respondWith400(w, r, ErrInvalidRegistrationKey.Error())
			return
		}

		splittedCachePayload := strings.Split(cachePayload, ";")
		if len(splittedCachePayload) != 2 {
			respondWith500(w, r, "")
			log.Printf("Expected two strings in the array after splitting the payload from cache, but received: %v", len(splittedCachePayload))
			return
		}

		emailAddress := splittedCachePayload[0]
		callbackUrl, err := url.Parse(splittedCachePayload[1])
		if err != nil {
			respondWith500(w, r, "")
			log.Printf("failed to parse callback url: %v", callbackUrl)
		}

		tx, err := db.BeginTx(r.Context(), nil)
		if err != nil {
			respondWith500(w, r, "")
			log.Println(err)
			return
		}

		defer func() {
			err = tx.Commit()
			if err != nil {
				log.Printf("error occured while trying to commit transaction: %v", err)
			}
		}()

		userId, err := users.Create(tx, emailAddress)
		if err != nil {
			txErr := tx.Rollback()
			if txErr != nil {
				log.Printf("failed to rollback the transaction: %v", txErr)
			}

			respondWith500(w, r, "")
			log.Printf("error occured while trying to create user in db: %v", err)
			return
		}

		value, err := oneTimeAccessTokenStore.PutAndGenerateRandomKeyForValue(fmt.Sprintf("userId:%d", userId))
		if err != nil {
			txErr := tx.Rollback()
			if txErr != nil {
				log.Printf("failed to rollback the transaction: %v", txErr)
			}

			respondWith500(w, r, "")
			log.Printf("error occured while trying to generate random access token: %v", err)
			return
		}

		queryParams := callbackUrl.Query()
		queryParams.Set("oneTimeAccessToken", value)

		callbackUrl.RawQuery = queryParams.Encode()

		w.Header().Add("Location", callbackUrl.String())
		w.WriteHeader(http.StatusTemporaryRedirect)

		_, err = w.Write([]byte("Przekierowywanie, proszę czekać..."))
		if err != nil {
			oneTimeAccessTokenStore.Delete(value)

			txErr := tx.Rollback()
			if txErr != nil {
				log.Printf("failed to rollback the transaction: %v", txErr)
			}

			respondWith500(w, r, "")
			log.Printf("error occured while trying to write response body to the client: %v", err)
			return
		}

		regkeyStore.Delete(regkey)
	}
}
