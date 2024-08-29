package server

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"net/mail"
	"net/url"

	"domanscy.group/parental-controls/server/shared"
)

func respondWith400(w http.ResponseWriter, _ *http.Request, message string) {
	if message == "" {
		message = "Bad Request"
	}

	w.WriteHeader(400)
	_, err := w.Write([]byte(message))
	if err != nil {
		log.Println("Error writing response:", err)
	}
}

func decodeJsonRequestBodyAndSendHttpErrorIfInvalid(w http.ResponseWriter, r *http.Request, decodedStruct interface{}) error {
	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(decodedStruct)
	if err != nil {
		respondWith400(w, r, shared.ErrInvalidJsonPayload.Error())
		return err
	}

	return nil
}

func parseEmailAddressAndHandleErrorIfInvalid(w http.ResponseWriter, r *http.Request, email string) error {
	address, err := mail.ParseAddress(email)
	if err != nil {
		respondWith400(w, r, shared.ErrInvalidEmail.Error())
		return err
	}

	if !(address.Name == "" && address.Address != "") {
		respondWith400(w, r, shared.ErrInvalidEmail.Error())
		return shared.ErrInvalidEmail
	}

	return nil
}

func parseUrlAndHandleErrorIfInvalid(w http.ResponseWriter, r *http.Request, value string) (*url.URL, error) {
	parsed, err := url.Parse(value)
	if err != nil {
		respondWith400(w, r, shared.ErrInvalidCallbackUrl.Error())
		return nil, err
	} else if parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		respondWith400(w, r, shared.ErrInvalidCallbackUrl.Error())
		return nil, shared.ErrInvalidCallbackUrl
	}

	return parsed, nil
}

func respondWith500(w http.ResponseWriter, _ *http.Request, message string) {
	if message == "" {
		message = "Internal Server Error"
	}

	w.WriteHeader(500)
	_, err := w.Write([]byte(message))
	if err != nil {
		log.Printf("Error responding with 500: %v", err)
	}
}

func getIPAddressFromRequest(_ http.ResponseWriter, req *http.Request) (string, error) {
	forward := req.Header.Get("X-Forwarded-For")
	ip, _, err := net.SplitHostPort(forward)
	if err == nil {
		return ip, nil
	}

	ip, _, err = net.SplitHostPort(req.RemoteAddr)
	if err == nil {
		return ip, nil
	}

	return "", errors.New("could not retrieve ip address from this request")
}
