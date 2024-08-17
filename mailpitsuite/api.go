package mailpitsuite

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Api struct {
	client   *http.Client
	baseUrl  string
	tenantId string
	command  *exec.Cmd
}

var mailpitMutex sync.Mutex

func NewApi(mailpitExecutablePath string) (*Api, error) {
	mailpitMutex.Lock()

	tenantId := fmt.Sprintf("tenant_%d", time.Now().UnixMilli())

	api := &Api{
		client:   &http.Client{},
		baseUrl:  "http://localhost:8025",
		tenantId: tenantId,
		command:  exec.Command(mailpitExecutablePath, "--tenant-id", tenantId),
	}

	api.command.Stderr = os.Stderr

	stdout, err := api.command.StdoutPipe()
	if err != nil {
		mailpitMutex.Unlock()
		return nil, fmt.Errorf("failed to obtain stdout pipe: %w", err)
	}

	stdoutReader := bufio.NewReader(stdout)

	err = api.command.Start()
	if err != nil {
		mailpitMutex.Unlock()
		return nil, fmt.Errorf("failed to start mailpit executable: %w", err)
	}

	// wait for http server in mailpit
	var allOutput strings.Builder

	for !strings.Contains(allOutput.String(), "level=info msg=\"[http] accessible via http://localhost:8025/\"") {
		tmp, err := stdoutReader.ReadString('\n')

		if err != nil && err != io.EOF {
			mailpitMutex.Unlock()
			killErr := api.command.Process.Kill()
			if killErr != nil {
				return nil, fmt.Errorf("failed to kill the process (%w) after failing to read stdout (%w)", killErr, err)
			}

			return nil, fmt.Errorf("failed to read stdout: %w", err)
		} else {
			allOutput.WriteString(tmp)
		}
	}

	for {
		_, err = http.Get(fmt.Sprintf("%s/api/v1/info", api.baseUrl))
		if err == nil {
			time.Sleep(500)
			break
		}

		time.Sleep(500)
	}

	return api, nil
}

func (api *Api) Close() error {
	mailpitMutex.Unlock()
	err := api.command.Process.Kill()
	if err != nil {
		return fmt.Errorf("failed to kill mailpit process: %w", err)
	}

	return nil
}

func (api *Api) DeleteAllMessages() error {
	requestUrl := fmt.Sprintf("%s/api/v1/messages", api.baseUrl)

	requestBody := "{\"IDs\": []}"

	req, err := http.NewRequest(http.MethodDelete, requestUrl, strings.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("an error occured while trying to create a new request: %w", err)
	}

	response, err := api.client.Do(req)
	if err != nil {
		return fmt.Errorf("an error occured while trying to send a request to mailpit api: %w", err)
	}

	if response.StatusCode != 200 {
		return fmt.Errorf("expected status 200 in response, received %d", response.StatusCode)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if string(responseBody) != ("ok") {
		return fmt.Errorf("response body from mailpit is not equal to 'ok', received %s", string(responseBody))
	}

	return nil
}

type Address struct {
	Address string `json:"Address"`
	Name    string `json:"Name"`
}

type Message struct {
	Attachments int       `json:"Attachments"`
	Bcc         []Address `json:"Bcc"`
	Cc          []Address `json:"Cc"`
	Created     time.Time `json:"Created"`
	From        Address   `json:"From"`
	ID          string    `json:"ID"`
	MessageID   string    `json:"MessageID"`
	Read        bool      `json:"Read"`
	ReplyTo     []Address `json:"ReplyTo"`
	Size        int       `json:"Size"`
	Snippet     string    `json:"Snippet"`
	Subject     string    `json:"Subject"`
	Tags        []string  `json:"Tags"`
	To          []Address `json:"To"`
}

type PaginatedMessages struct {
	Messages      []Message `json:"messages"`
	MessagesCount int       `json:"messages_count"`
	Start         int       `json:"start"`
	Tags          []string  `json:"tags"`
	Total         int       `json:"total"`
	Unread        int       `json:"unread"`
}

func (api *Api) GetAllMessages() ([]Message, error) {
	type RequestBody struct {
		Start int `json:"start"`
		Limit int `json:"limit"`
	}

	total := 999999999
	start := 0

	allMessages := make([]Message, 0)

	for start < total {
		body := RequestBody{
			Start: 0,
			Limit: 50,
		}

		requestBodyBuffer, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to convert request body to json payload: %w", err)
		}

		request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/messages", api.baseUrl), bytes.NewReader(requestBodyBuffer))
		if err != nil {
			return nil, fmt.Errorf("error occured while creating request object: %w", err)
		}

		response, err := api.client.Do(request)
		if err != nil {
			return nil, fmt.Errorf("an error occured while trying to send the request to mailpit: %w", err)
		}

		responseBodyBuffer, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response buffer: %w", err)
		}

		defer response.Body.Close()

		if response.StatusCode != 200 {
			responseBodyStringBuffer := string(responseBodyBuffer)
			return nil, fmt.Errorf("expected status 200, received status %d, request body: %s", response.StatusCode, responseBodyStringBuffer)
		}

		var paginatedMessages = &PaginatedMessages{}

		err = json.Unmarshal(responseBodyBuffer, paginatedMessages)
		if err != nil {
			return nil, fmt.Errorf("failed to parse json response: %w", err)
		}

		// set total at the first iteration
		if total == 999999999 {
			total = paginatedMessages.Total
		}

		start += paginatedMessages.MessagesCount

		allMessages = append(allMessages, paginatedMessages.Messages...)
	}

	return allMessages, nil
}
