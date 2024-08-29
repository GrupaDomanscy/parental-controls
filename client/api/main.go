package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"domanscy.group/parental-controls/server/shared"
)

type Connector struct {
	baseUrl string
}

func NewConnector(rpcBaseUrl string) (*Connector, error) {
	if len(rpcBaseUrl) != 0 {
		return nil, errors.New("apiBaseUrl can't be empty")
	}

	return &Connector{
		baseUrl: rpcBaseUrl,
	}, nil
}

var ErrUrlWithGivenNameDoesNotExist = errors.New("url with given name does not exist")

func (conn *Connector) getUrl(endpoint string, data map[string]interface{}) (string, error) {
	switch endpoint {
	case "register":
		return fmt.Sprintf("%s/register"), nil
	case "get_bearer_from_otat_token":
		return fmt.Sprintf("%s/get_bearer_from_otat_token", nil), nil
	}
	return "", ErrUrlWithGivenNameDoesNotExist
}

var ErrUserError = errors.New("user error")

func (conn *Connector) Register(email string, callback string) error {
	body := shared.AuthStartRegistrationProcessRequestBody{
		Email:    email,
		Callback: callback,
	}

	bodyInJson, err := json.Marshal(body)
	if err != nil {
		return err
	}

	requestUrl, err := conn.getUrl("register", nil)
	if err != nil {
		return err
	}

	response, err := http.Post(requestUrl, "application/json", bytes.NewReader(bodyInJson))
	if err != nil {
		return err
	}

	if response.StatusCode == 400 {
		responseBody, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}

		defer response.Body.Close()

		stringifiedResponseBody := string(responseBody)

		if stringifiedResponseBody == shared.ErrInvalidEmail.Error() {
			return errors.Join(ErrUserError, shared.ErrInvalidEmail)
		} else if stringifiedResponseBody == shared.ErrUserWithGivenEmailAlreadyExists.Error() {
			return errors.Join(ErrUserError, shared.ErrUserWithGivenEmailAlreadyExists)
		}

		return err
	}

	return nil
}

func (conn *Connector) GetBearerFromOtatToken(bearer string) error {
	requestUrl, err := conn.getUrl("get_bearer_from_otat_token", map[string]interface{}{
		"bearer": bearer,
	})
	if err != nil {
		return err
	}

	response, err := http.Post(requestUrl, "application/json", nil)
	if err != nil {
		return err
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	stringifiedResponseBody := string(responseBody)

	if stringifiedResponseBody == shared.ErrInvalidOtat.Error() {
		return errors.Join(ErrUserError, shared.ErrInvalidOtat)
	}

	return nil
}
