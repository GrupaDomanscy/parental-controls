package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
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

func usage() {
	fmt.Printf("Usage: %s [OPTIONS] argument ...\n", os.Args[0])
	fmt.Println()
	fmt.Println("Arguments:")
	fmt.Println("  generate-private-key - generates private key to stdout (there is also an option to write to the file directly, see -output)")
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	command = flag.Arg(0)

	if command == "generate-private-key" {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			log.Fatalf("error occured while trying to generate private key: %v", err)
		}

		privateKeyInBytes := x509.MarshalPKCS1PrivateKey(privateKey)
		privateKeyInBase64 := hex.EncodeToString(privateKeyInBytes)

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
		flag.Usage()
	}
}
