package main

import (
	"archive/zip"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func must[T any](something T, err error) T {
	if err != nil {
		log.Fatal(err)
	}

	return something
}

func getCwd() string {
	return must(os.Getwd())
}

var TmpDirectory = filepath.Join(getCwd(), "tmp")
var MailpitDownloadLink = "https://github.com/axllent/mailpit/releases/download/v1.20.1/mailpit-windows-amd64.zip"
var MailpitDownloadedZipOutputFileName = filepath.Join(TmpDirectory, "mailpit.zip")
var MailpitDownloadedExeOutputFileName = filepath.Join(TmpDirectory, "mailpit.exe")

func init() {
	_, err := os.Stat(MailpitDownloadedExeOutputFileName)
	if err == nil {
		log.Println("Mailpit already on the machine! Launching tests...")
		return
	}

	if _, err = os.Stat(TmpDirectory); err == nil {
		err = os.Remove(TmpDirectory)
		if err != nil {
			log.Fatalf("Failed to remove tmp directory: %v", err)
		}
	}

	err = os.Mkdir("tmp", 0750)
	if err != nil {
		log.Fatalf("Failed to create tmp directory: %v", err)
	}

	log.Println("Downloading mailpit...")

	zipFile, err := os.Create(MailpitDownloadedZipOutputFileName)
	if err != nil {
		log.Fatalf("Failed to create a file for a download: %v", err)
	}

	defer zipFile.Close() // close might be called two times, ignore it.

	response, err := http.Get(MailpitDownloadLink)
	if err != nil {
		log.Fatalf("Failed to download a zip file with mailpit app: %v", err)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatalf("Failed to close a http response: %v", err)
		}
	}(response.Body)

	_, err = io.Copy(zipFile, response.Body)
	if err != nil {
		log.Fatalf("Failed to save a downloaded payload to a file: %v", err)
	}

	err = zipFile.Close()
	if err != nil {
		log.Fatalf("Failed to close a zip file %v: ", err)
	}

	log.Println("Decompressing the zip file.")

	zipReader, err := zip.OpenReader(MailpitDownloadedZipOutputFileName)
	if err != nil {
		log.Fatalf("Failed to open a zip reader on a downloaded mailpit zip: %v", err)
	}

	defer zipReader.Close()

	exeFile, err := os.Create(MailpitDownloadedExeOutputFileName)
	if err != nil {
		log.Fatalf("Failed to create mailpit exe file: %v", err)
	}

	for _, zippedFile := range zipReader.File {
		if zippedFile.Name != "mailpit.exe" {
			continue
		}

		zippedMailpitFileReader, err := zippedFile.Open()
		if err != nil {
			log.Fatal(err)
		}

		defer zippedMailpitFileReader.Close()

		_, err = io.Copy(exeFile, zippedMailpitFileReader)
		if err != nil {
			log.Fatal(err)
		}

		break
	}
}
