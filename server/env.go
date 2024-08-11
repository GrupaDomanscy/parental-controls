package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

var errDotenvFileDoesNotExist = errors.New("file .env has not been found on the system")

func readEnvironmentVarsFromDotenvFile() (map[string]string, error) {
	file, err := os.ReadFile(".env")
	if err != nil {
		return nil, errDotenvFileDoesNotExist
	}

	contents := string(file)

	// remove all \r chars
	contents = strings.Join(strings.Split(contents, "\r"), "")

	lines := strings.Split(contents, "\n")

	return readEnvironmentVarsFromPayload(lines)
}

func readEnvironmentVarsFromPayload(payload []string) (map[string]string, error) {
	var envVars map[string]string = make(map[string]string)

	for i, line := range payload {
		parts := strings.Split(line, "=")
		if len(parts) == 0 && len(line) != 0 {
			return nil, errors.New(fmt.Sprintf("Invalid format in line %d No equal sign detected.\n", i+1))
		}

		if len(parts) == 1 {
			return nil, errors.New(fmt.Sprintf("Invalid format in line %d.", i+1))
		}

		envVars[parts[0]] = strings.Join(parts[1:], "")
	}

	return envVars, nil
}

func readEnvironmentVarsFromShell() (map[string]string, error) {
	return readEnvironmentVarsFromPayload(os.Environ())
}
