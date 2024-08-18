package main

import (
	"bytes"
	"database/sql"
	"domanscy.group/parental-controls/server/regkeys"
	"domanscy.group/parental-controls/server/users"
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

func HttpAuthLogin(cfg *ServerConfig, _ *regkeys.Store, db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
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

		user, err := users.FindOneByEmail(db, requestBody.Email)
		if err != nil {
			respondWith500(w, r, "")
			log.Printf("error occured while trying to find user by email: %v", err)
			return
		}

		if user == nil {
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

func HttpAuthStartRegistrationProcess(cfg *ServerConfig, regkeysStore *regkeys.Store, db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
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

		user, err := users.FindOneByEmail(db, requestBody.Email)
		if err != nil {
			respondWith500(w, r, "")
			log.Printf("error occured while trying to find user by email: %v", err)
			return
		}

		if user != nil {
			respondWith400(w, r, ErrUserWithGivenEmailAlreadyExists.Error())
			return
		}

		regkey, err := regkeysStore.GenerateNewRegkeyForEmail(requestBody.Email)
		if err != nil {
			respondWith500(w, r, "")
			log.Printf("an error occured while trying to generate new regkey for email '%s': %v", requestBody.Email, err)
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
			respondWith500(w, r, "")
			log.Printf("an errorm occured while trying to generate email body: %v", err)
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
			log.Println(err)
			respondWith500(w, r, "")
			return
		}

		w.WriteHeader(204)
	}
}
