package server

import (
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"strings"
)

func sendMailAndHandleError(
	w http.ResponseWriter,
	r *http.Request,
	smtpAddress string,
	smtpPort uint16,
	fromAddress string,
	toAddress string,
	subject string,
	body string,
) error {
	var message strings.Builder

	message.WriteString(fmt.Sprintf("From: %s\r\n", fromAddress))
	message.WriteString(fmt.Sprintf("To: %s\r\n", toAddress))
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	message.WriteString(fmt.Sprintf("Content-Type: text/html; charset=utf-8\r\n"))
	message.WriteString("\r\n")
	message.WriteString(body)

	err := smtp.SendMail(
		fmt.Sprintf("%s:%d", smtpAddress, smtpPort),
		nil,
		fromAddress,
		[]string{toAddress},
		[]byte(message.String()),
	)
	if err != nil {
		log.Println(err)
		respondWith500(w, r, "")
		return err
	}

	return nil
}
