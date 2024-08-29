package server

import (
	"bytes"
	"io"
	"log"
	"mailpitsuite"
	"os"
	"path/filepath"
	"runtime"
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

func getMailpitExecutableFilePath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(getCwd(), "tmp", "mailpit.exe")
	} else if runtime.GOOS == "linux" {
		return filepath.Join(getCwd(), "tmp", "mailpit")
	}

	panic("mailpitsuite supports linux and windows only")
}

func init() {
	log.SetFlags(log.Ldate | log.LUTC | log.Lmicroseconds | log.Llongfile)
	mailpitExecutablePath := getMailpitExecutableFilePath()

	_, err := os.Stat(mailpitExecutablePath)
	if err == nil {
		log.Println("mailpit is already downloaded on this machine")
		return
	}

	mailpitExeParentDirPath := filepath.Dir(mailpitExecutablePath)

	if _, err = os.Stat(mailpitExeParentDirPath); err != nil {
		err := os.MkdirAll(mailpitExeParentDirPath, 0741)
		if err != nil {
			log.Fatal(err)
		}
	}

	mailpitExeFile, err := os.Create(mailpitExecutablePath)
	if err != nil {
		log.Fatal(err)
	}

	defer func(mailpitExeFile *os.File) {
		logFatalIfErr(mailpitExeFile.Close())
	}(mailpitExeFile)

	err = os.Chmod(mailpitExecutablePath, 0766)
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
