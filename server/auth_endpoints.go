package main

import (
	"bytes"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"domanscy.group/littlehelpers"
	"domanscy.group/parental-controls/server/users"
	"domanscy.group/rckstrvcache"
	"github.com/go-chi/chi"
)

var ErrUserWithGivenEmailDoesNotExist = errors.New("user with given email does not exist")

func HttpAuthLogin(cfg *ServerConfig, _ *rckstrvcache.Store, _ *rckstrvcache.Store, db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
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

func HttpAuthStartRegistrationProcess(cfg *ServerConfig, regkeysStore *rckstrvcache.Store, _ *rckstrvcache.Store, db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
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

		user, err := users.FindOneByEmail(tx, requestBody.Email)
		if err != nil {
			txErr := tx.Rollback()
			if txErr != nil {
				err = errors.Join(err, fmt.Errorf("failed to rollback the transaction: %v", txErr))
			}

			respondWith500(w, r, "")
			log.Printf("error occured while trying to find user by email: %v", err)
			return
		}

		if user != nil {
			txErr := tx.Rollback()
			if txErr != nil {
				err = errors.Join(err, fmt.Errorf("failed to rollback the transaction: %v", txErr))
			}

			respondWith400(w, r, ErrUserWithGivenEmailAlreadyExists.Error())
			return
		}

		err = regkeysStore.InTransaction(func(regkeysTx rckstrvcache.StoreCompatible) error {
			regkey, err := regkeysTx.Put(requestBody.Email + ";" + callbackUrl.String())
			if err != nil {
				return fmt.Errorf("an error occured while trying to generate new regkey for email '%s': %v", requestBody.Email, err)
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
				return fmt.Errorf("failed to construct email template: %w", err)
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
				return fmt.Errorf("failed to send mail: %w", err)
			}

			return nil
		})
		if err != nil {
			txErr := tx.Rollback()
			if txErr != nil {
				err = errors.Join(err, fmt.Errorf("failed to rollback the transaction: %v", txErr))
			}

			respondWith500(w, r, "")
			log.Println(err)
			return
		}

		err = tx.Commit()
		if err != nil {
			log.Printf("failed to commit the transaction: %v", err)
			respondWith500(w, r, "")
			return
		}

		w.WriteHeader(204)
	}
}

var ErrRegistrationKeyCannotBeEmpty = errors.New("registration key can not be empty")
var ErrInvalidRegistrationKey = errors.New("invalid registration key")

func HttpAuthFinishRegistrationProcess(_ *ServerConfig, regkeyStore *rckstrvcache.Store, oneTimeAccessTokenStore *rckstrvcache.Store, db *sql.DB) http.HandlerFunc {
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

		var callbackUrl *url.URL

		err = regkeyStore.InTransaction(func(regkeyStore rckstrvcache.StoreCompatible) error {
			cachePayload, exists, err := regkeyStore.Get(regkey)
			if err != nil {
				return fmt.Errorf("error occured while trying to get information about regkey from database: %w", err)
			}

			if !exists {
				return ErrInvalidRegistrationKey
			}

			splittedCachePayload := strings.Split(cachePayload, ";")
			if len(splittedCachePayload) != 2 {
				return fmt.Errorf("expected two strings in the array after splitting the payload from cache, but received: %v", len(splittedCachePayload))
			}

			emailAddress := splittedCachePayload[0]
			callbackUrl, err = url.Parse(splittedCachePayload[1])
			if err != nil {
				return fmt.Errorf("failed to parse callback url: %v", callbackUrl)
			}

			tx, err := db.BeginTx(r.Context(), nil)
			if err != nil {
				return fmt.Errorf("failed to open database transaction: %w", err)
			}

			userId, err := users.Create(tx, emailAddress)
			if err != nil {
				txErr := tx.Rollback()
				if txErr != nil {
					err = errors.Join(err, txErr)
				}

				return fmt.Errorf("error occured while trying to create user in db: %v", err)
			}

			err = oneTimeAccessTokenStore.InTransaction(func(oneTimeAccessTokenStore rckstrvcache.StoreCompatible) error {
				oneTimeAccessToken, err := oneTimeAccessTokenStore.Put(fmt.Sprintf("userId:%d", userId))
				if err != nil {
					return err
				}

				if err != nil {
					txErr := tx.Rollback()
					if txErr != nil {
						err = errors.Join(err, txErr)
					}

					return fmt.Errorf("error occured while trying to generate random access token: %w", err)
				}

				queryParams := callbackUrl.Query()
				queryParams.Set("oneTimeAccessToken", oneTimeAccessToken)

				callbackUrl.RawQuery = queryParams.Encode()

				_, err = regkeyStore.Delete(regkey)
				if err != nil {
					return fmt.Errorf("error occured while trying to remove regkey from cache: %w", err)
				}

				return nil
			})
			if err != nil {
				txErr := tx.Rollback()
				if txErr != nil {
					err = errors.Join(err, txErr)
				}
				return err
			}

			err = tx.Commit()
			if err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			if errors.Is(err, ErrInvalidRegistrationKey) {
				respondWith400(w, r, err.Error())
				return
			}

			log.Println(err)
			respondWith500(w, r, "")
			return
		}

		w.Header().Add("Location", callbackUrl.String())
		w.WriteHeader(http.StatusTemporaryRedirect)
	}
}

var ErrInvalidOtat = errors.New("invalid one time access token")

func HttpAuthGetBearerTokenFromOtat(cfg *ServerConfig, regkeyStore *rckstrvcache.Store, otatStore *rckstrvcache.Store, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		otatToken := chi.URLParam(r, "otat")

		if len(otatToken) == 0 {
			respondWith400(w, r, ErrInvalidOtat.Error())
			return
		}

		otatTx, err := otatStore.Begin()

		payload, exists, err := otatTx.Get(otatToken)
		if err != nil {
			err = littlehelpers.IfErrJoin(otatTx.Rollback(), err)
			log.Printf("error occured while trying to get otat from cache: %v", err)
			respondWith500(w, r, "")
			return
		}

		if !exists {
			err = otatTx.Rollback()
			if err != nil {
				respondWith500(w, r, "")
				log.Println(err)
				return
			}

			respondWith400(w, r, ErrInvalidOtat.Error())
			return
		}

		userId, err := strconv.Atoi(strings.Replace(payload, "userId:", "", 1))
		if err != nil {
			err = littlehelpers.IfErrJoin(err, otatTx.Rollback())
			log.Printf("error occured while trying to get userId from otat cache payload: %v", err)
			respondWith500(w, r, "")
			return
		}

		tx, err := db.Begin()
		if err != nil {
			err = littlehelpers.IfErrJoin(err, otatTx.Rollback())
			log.Printf("error occured while trying to start transaction: %v", err)
			respondWith500(w, r, "")
			return
		}

		user, err := users.FindOneById(tx, userId)
		if err != nil {
			err = littlehelpers.IfErrJoin(err, otatTx.Rollback())
			log.Printf("error occured while trying to find user by id: %v", err)
			respondWith500(w, r, "")
			return
		}

		bearer, err := CreateBearerTokenForUser(cfg.BearerTokenPrivateKey, user.Id)
		if err != nil {
			err = littlehelpers.IfErrJoin(err, otatTx.Rollback())
			log.Printf("error occured while trying to create bearer token: %v", err)
			respondWith500(w, r, "")
			return
		}

		err = tx.Commit()
		if err != nil {
			err = littlehelpers.IfErrJoin(err, otatTx.Rollback())
			log.Printf("error occured while trying to commit: %v", err)
			respondWith500(w, r, "")
			return
		}

		err = otatTx.Commit()
		if err != nil {
			log.Printf("error occured while trying to commit: %v", err)
			respondWith500(w, r, "")
			return
		}

		w.WriteHeader(200)
		w.Write(bearer)
	}
}
