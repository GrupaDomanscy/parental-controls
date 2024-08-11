package main

import (
	"fmt"
	"net/smtp"
)

func NewSmtpConnection(address string, port uint16) *smtp.Client {
	client, err := smtp.Dial(fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		panic(err)
	}

	return client
}
