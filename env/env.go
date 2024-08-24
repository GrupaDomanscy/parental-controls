package env

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"math"
	"net/url"
	"os"
	"strconv"
)

func ParseStringVar(envName string) (value string, exists bool) {
	return os.LookupEnv(envName)
}

func ParseIntVar(envName string) (value int, exists bool, err error) {
	valueInString, exists := os.LookupEnv(envName)
	if !exists {
		return 0, false, nil
	}

	value, err = strconv.Atoi(valueInString)
	if err != nil {
		return 0, true, fmt.Errorf("env '%s' must be a valid integer", envName)
	}

	return value, true, nil
}

func ParsePrivateKeyVarFromFilePath(envName string) (value *rsa.PrivateKey, exists bool, err error) {
	pathStr, exists := os.LookupEnv(envName)
	if !exists {
		return nil, false, nil
	}

	rawFile, err := os.ReadFile(pathStr)
	if err != nil {
		return nil, true, fmt.Errorf("failed to read file %s", err)
	}

	decodedHex := make([]byte, hex.DecodedLen(len(rawFile)))

	_, err = hex.Decode(decodedHex, rawFile)
	if err != nil {
		return nil, true, fmt.Errorf("env '%s' must be a valid private key converted to hex: %w", envName, err)
	}

	value, err = x509.ParsePKCS1PrivateKey(decodedHex)
	if err != nil {
		return nil, true, fmt.Errorf("env '%s' must be a valid private key converted to hex: %w", envName, err)
	}

	return value, true, nil
}

func ParsePrivateKeyVar(envName string) (value *rsa.PrivateKey, exists bool, err error) {
	rawValue, exists := os.LookupEnv(envName)
	if !exists {
		return nil, false, nil
	}

	var decodedHex []byte

	_, err = hex.Decode(decodedHex, []byte(rawValue))
	if err != nil {
		return nil, true, fmt.Errorf("env '%s' must be a valid private key converted to hex: %w", envName, err)
	}

	value, err = x509.ParsePKCS1PrivateKey(decodedHex)
	if err != nil {
		return nil, true, fmt.Errorf("env '%s' must be a valid private key converted to hex: %w", envName, err)
	}

	return value, true, nil
}

func ParseValidUrlVarWithHttpOrHttpsProtocol(envName string) (value *url.URL, exists bool, err error) {
	rawValue, exists := os.LookupEnv(envName)
	if !exists {
		return nil, false, nil
	}

	parsedUrl, err := url.Parse(rawValue)
	if err != nil {
		return nil, true, err
	}

	if parsedUrl.Scheme != "https" && parsedUrl.Scheme != "http" {
		return nil, true, fmt.Errorf("invalid scheme in url, required http or https")
	}

	return parsedUrl, true, nil
}

func ParseUint16Var(envName string) (value uint16, valid bool, err error) {
	rawValue, exists := os.LookupEnv(envName)
	if !exists {
		return 0, false, nil
	}

	parsedWithoutCast, err := strconv.ParseUint(rawValue, 10, 16)
	if err != nil {
		return 0, true, err
	}

	if parsedWithoutCast < 0 || parsedWithoutCast > math.MaxUint16 {
		return 0, true, fmt.Errorf("not a valid uint16")
	}

	parsedWithCast := uint16(parsedWithoutCast)

	return parsedWithCast, true, nil
}
