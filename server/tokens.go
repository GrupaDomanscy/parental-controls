package server

import (
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"domanscy.group/parental-controls/server/encryption"
)

func CreateBearerTokenForUser(privateKey *rsa.PrivateKey, userId int) ([]byte, error) {
	payload := fmt.Sprintf("userId:%d", userId)

	encrypted, err := encryption.Encrypt(privateKey, []byte(payload))
	if err != nil {
		return nil, fmt.Errorf("error occured while encrypting token payload: %w", err)
	}

	hexEncoded := []byte(hex.EncodeToString(encrypted))

	return hexEncoded, nil
}

func GetUserIdFromBearerToken(privateKey *rsa.PrivateKey, token []byte) (int, error) {
	hexDecoded, err := hex.DecodeString(string(token))
	if err != nil {
		return 0, err
	}

	decrypted, err := encryption.Decrypt(privateKey, []byte(hexDecoded))
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
