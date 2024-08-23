package main

import (
	"crypto/rsa"
	"domanscy.group/parental-controls/server/encryption"
	"fmt"
	"strconv"
	"strings"
)

func CreateBearerTokenForUser(privateKey *rsa.PrivateKey, userId int) ([]byte, error) {
	payload := fmt.Sprintf("userId:%d", userId)

	encrypted, err := encryption.Encrypt(privateKey, []byte(payload))
	if err != nil {
		return nil, fmt.Errorf("error occured while encrypting token payload: %w", err)
	}

	return encrypted, nil
}

func GetUserIdFromBearerToken(privateKey *rsa.PrivateKey, token []byte) (int, error) {
	decrypted, err := encryption.Decrypt(privateKey, token)
	if err != nil {
		return 0, err
	}

	prefix := "userId:"

	if !strings.HasPrefix(string(decrypted), prefix) {
		return 0, fmt.Errorf("decrypted token is in invalid format, expected to start with userId")
	}

	userId, err := strconv.Atoi(string(decrypted)[len(prefix):])
	if err != nil {
		return 0, fmt.Errorf("error occured while trying to get userId from token")
	}

	return userId, nil
}
