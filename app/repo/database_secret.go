package repo

import (
	"fmt"
	"strings"

	"github.com/aihop/gopanel/utils/encrypt"
)

const databaseSecretPrefix = "enc:v1:"

func encryptDatabaseSecret(value string) (string, error) {
	if value == "" || strings.HasPrefix(value, databaseSecretPrefix) {
		return value, nil
	}
	ciphertext, err := encrypt.StringEncrypt(value)
	if err != nil {
		return "", fmt.Errorf("encrypt database credential: %w", err)
	}
	return databaseSecretPrefix + ciphertext, nil
}

func decryptDatabaseSecret(value string) (string, error) {
	if value == "" || !strings.HasPrefix(value, databaseSecretPrefix) {
		return value, nil
	}
	plaintext, err := encrypt.StringDecrypt(strings.TrimPrefix(value, databaseSecretPrefix))
	if err != nil {
		return "", fmt.Errorf("decrypt database credential: %w", err)
	}
	return plaintext, nil
}
