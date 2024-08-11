package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"net/smtp"
	"strings"
)

func HttpAuthLogin(cfg *ServerConfig, db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		type RequestBody struct {
			Email string `json:"email"`
		}

		decoder := json.NewDecoder(r.Body)
		var requestBody RequestBody

		err := decoder.Decode(&requestBody)
		if err != nil {
			log.Printf("Error decoding body: %v", err)
			err := respondWith400(w, r, ErrInvalidJsonPayload.Error())
			if err != nil {
				log.Println("Error responding with 400:", err)
			}

			return
		}

		requestBody.Email = strings.ToLower(requestBody.Email)

		address, err := mail.ParseAddress(requestBody.Email)
		if err != nil || !(address.Name == "" && address.Address != "") {
			err = respondWith400(w, r, ErrInvalidEmail.Error())
			log.Println("Error responding with 400:", err)
			return
		}

		var message strings.Builder

		message.WriteString(fmt.Sprintf("From: %s\r\n", cfg.EmailFromAddress))
		message.WriteString(fmt.Sprintf("To: %s\r\n", address.Address))
		message.WriteString(fmt.Sprintf("Subject: %s\r\n", "Zaloguj się"))
		message.WriteString(fmt.Sprintf("\r\n"))
		message.WriteString(fmt.Sprintf("Hello world!\r\n"))

		err = smtp.SendMail(
			fmt.Sprintf("%s:%d", cfg.SmtpAddress, cfg.SmtpPort),
			nil,
			cfg.EmailFromAddress,
			[]string{address.Address},
			[]byte(message.String()),
		)
		if err != nil {
			return
		}
	}
}
