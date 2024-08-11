package main

import (
	"errors"
	"net/http"
)

var ErrInvalidJsonPayload = errors.New("Invalid json payload")
var ErrInvalidEmail = errors.New("Invalid email")

func respondWith400(w http.ResponseWriter, r *http.Request, message string) error {
	if message == "" {
		message = "Bad Request"
	}

	w.WriteHeader(400)
	_, err := w.Write([]byte(message))
	if err != nil {
		return err
	}

	return nil
}
