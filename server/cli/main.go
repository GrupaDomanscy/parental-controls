package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
)

var command string
var output string

func init() {
	flag.StringVar(&output, "output", "", "Output file location, valid for commands: generate-private-key. If output is not supplied, it will write to stdout.")
}

func main() {
	flag.Parse()

	command = flag.Arg(0)

	if command == "generate-private-key" {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			log.Fatalf("error occured while trying to generate private key: %v", err)
		}

		privateKeyInBytes := x509.MarshalPKCS1PrivateKey(privateKey)
		privateKeyInBase64 := base64.StdEncoding.EncodeToString(privateKeyInBytes)

		if output == "" {
			fmt.Println(privateKeyInBase64)
		} else {
			os.WriteFile(output, []byte(privateKeyInBase64), 0666)
			if err != nil {
				log.Fatalf("failed to write private key to file '%s' with permissions: %d", output, 0666)
			}
		}
	} else {
		fmt.Println("unknown command supplied")
		fmt.Println("valid commands: generate-private-key")
	}
}
