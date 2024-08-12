package mailpitsuite

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
)

var mailpitDownloadLink = "https://github.com/axllent/mailpit/releases/download/v1.20.1/mailpit-windows-amd64.zip"

func downloadMailpitZip() ([]byte, error) {
	response, err := http.Get(mailpitDownloadLink)
	if err != nil {
		return nil, fmt.Errorf("failed to download a zip file with mailpit app: %w", err)
	}
	defer response.Body.Close()

	var fileBuffer = bytes.NewBuffer([]byte{})

	_, err = io.Copy(fileBuffer, response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to extract downloaded file: %w", err)
	}

	return fileBuffer.Bytes(), nil
}

func getMailpitExecutableFromZipBuffer(zipFile []byte) (mailpitExecutable []byte, err error) {
	var zipFileBuffer = bytes.NewReader(zipFile)

	reader, err := zip.NewReader(zipFileBuffer, int64(len(zipFile)))
	if err != nil {
		return nil, fmt.Errorf("failed to open a zip reader: %w", err)
	}

	file, err := reader.Open("mailpit.exe")
	if err != nil {
		return nil, fmt.Errorf("failed to open an executable file in zip: %w", err)
	}

	fileContents := bytes.NewBuffer(mailpitExecutable)

	_, err = io.Copy(fileContents, file)
	if err != nil {
		return nil, fmt.Errorf("an error occured while trying to read executable file contents from zip: %w", err)
	}

	return fileContents.Bytes(), nil
}

func Download() (executableContents []byte, err error) {
	zipfile, err := downloadMailpitZip()
	if err != nil {
		return nil, fmt.Errorf("failed to download mailpit zip: %w", err)
	}

	buffer, err := getMailpitExecutableFromZipBuffer(zipfile)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack mailpit executable from downloaded zip file: %w", err)
	}

	return buffer, nil
}
