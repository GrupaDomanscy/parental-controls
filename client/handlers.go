package client

import (
	"fmt"
	"log"
	"net/http"
	"net/smtp"

	"domanscy.group/parental-controls/client/components"
)

func GetUrl(opts ServerOpts) func(string, map[string]interface{}) string {
	return func(name string, args map[string]interface{}) string {
		switch name {
		case "register":
			return fmt.Sprintf("%s/register", opts.baseUrl)
		default:
			return ""
		}
	}
}

func RenderLoginPageHttpHandler(opts ServerOpts) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(components.LoginPage(GetUrl(opts)).Render()))
		if err != nil {
			log.Printf("Error occured while trying to write body data: %s\n", err.Error())
		}
	}
}

func LoginHttpHandler(opts ServerOpts) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		client, err := smtp.Dial(fmt.Sprintf("%s:%d", opts.smtpAddress, opts.smtpPort))
		defer func() {
			err := client.Close()
			if err != nil {
				log.Printf("Error occured while trying to close the connection with SMTP Server: %s\n", err.Error())
			}
		}()

		if err != nil {
			log.Printf("Error occured while trying to connect to SMTP Server: %s\n", err.Error())
			respondWith500(w, r)
			return
		}
	}
}
