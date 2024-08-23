package mailpitsuite

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"runtime"
)

const (
	mailpitWindowsExecutableName = "mailpit.exe"
	mailpitLinuxExecutableName   = "mailpit"
	mailpitWindowsDownloadLink   = "https://github.com/axllent/mailpit/releases/download/v1.20.1/mailpit-windows-amd64.zip"
	mailpitLinuxDownloadLink     = "https://github.com/axllent/mailpit/releases/download/v1.20.2/mailpit-linux-amd64.tar.gz"
)

func downloadMailpit() ([]byte, error) {
	var downloadLink string

	if runtime.GOOS == "windows" {
		downloadLink = mailpitWindowsDownloadLink
	} else if runtime.GOOS == "linux" {
		downloadLink = mailpitLinuxDownloadLink
	} else {
		panic("mailpitsuite only supports linux and windows")
	}

	response, err := http.Get(downloadLink)
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

func getMailpitExecutableFromGzipBuffer(gzipFile []byte) (mailpitExecutable []byte, err error) {
	var gzipFileReader = bytes.NewReader(gzipFile)

	gzipReader, err := gzip.NewReader(gzipFileReader)
	if err != nil {
		return nil, fmt.Errorf("failed to open a zip reader: %w", err)
	}

	reader := tar.NewReader(gzipReader)

	for true {
		fileHeader, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("mailpit has not been found in tar archive: io.EOF error")
			}
			return nil, fmt.Errorf("failed to read header data of a file: %w", err)
		}

		if fileHeader.Name != mailpitLinuxExecutableName {
			continue
		}

		fileContents := bytes.NewBuffer(mailpitExecutable)

		_, err = io.Copy(fileContents, reader)
		if err != nil {
			return nil, fmt.Errorf("an error occured while trying to read executable file contents from zip: %w", err)
		}

		return fileContents.Bytes(), nil
	}

	return nil, fmt.Errorf("mailpit has not been found in tar archive")
}

func getMailpitExecutableFromZipBuffer(zipFile []byte) (mailpitExecutable []byte, err error) {
	var zipFileBuffer = bytes.NewReader(zipFile)

	reader, err := zip.NewReader(zipFileBuffer, int64(len(zipFile)))
	if err != nil {
		return nil, fmt.Errorf("failed to open a zip reader: %w", err)
	}

	file, err := reader.Open(mailpitWindowsExecutableName)
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
	archiveBytes, err := downloadMailpit()
	if err != nil {
		return nil, fmt.Errorf("failed to download mailpit: %w", err)
	}

	var buffer []byte

	if runtime.GOOS == "windows" {
		buffer, err = getMailpitExecutableFromZipBuffer(archiveBytes)
	} else if runtime.GOOS == "linux" {
		buffer, err = getMailpitExecutableFromGzipBuffer(archiveBytes)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to unpack mailpit executable from downloaded archive: %w", err)
	}

	return buffer, nil
}
