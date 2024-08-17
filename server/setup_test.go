package main

import (
	"bytes"
	"io"
	"log"
	"mailpitsuite"
	"os"
	"path/filepath"
)

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func mustVal[T any](something T, err error) T {
	if err != nil {
		log.Fatal(err)
	}

	return something
}

func getCwd() string {
	return mustVal(os.Getwd())
}

var mailpitExeFilePath = filepath.Join(getCwd(), "tmp", "mailpit.exe")

func init() {
	_, err := os.Stat(mailpitExeFilePath)
	if err == nil {
		log.Println("mailpit is already downloaded on this machine")
		return
	}

	mailpitExeParentDirPath := filepath.Dir(mailpitExeFilePath)

	if _, err = os.Stat(mailpitExeParentDirPath); err != nil {
		err := os.MkdirAll(mailpitExeParentDirPath, 0660)
		if err != nil {
			log.Fatal(err)
		}
	}

	mailpitExeFile, err := os.Create(mailpitExeFilePath)
	if err != nil {
		log.Fatal(err)
	}

	mailpitExeFileContent, err := mailpitsuite.Download()
	if err != nil {
		log.Fatal(err)
	}

	_, err = io.Copy(mailpitExeFile, bytes.NewReader(mailpitExeFileContent))
	if err != nil {
		log.Fatal(err)
	}

	log.Println("mailpit has been successfully installed")
}
