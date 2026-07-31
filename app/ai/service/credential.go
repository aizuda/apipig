package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"

	"apipig/core/api/response"
)

const (
	maskedCredential = "********"
	tokenHashPrefix  = "sha256:"
)

var sensitiveHeaderPattern = regexp.MustCompile(`(?i)(authorization\s*[:=]\s*bearer\s+|x-api-key\s*[:=]\s*|api[_-]?key\s*[:=]\s*)[^\s,;]+`)

func generateSecureGatewayToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return "sk-" + base64.RawURLEncoding.EncodeToString(buffer), nil
}

func hashGatewayToken(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return tokenHashPrefix + hex.EncodeToString(sum[:])
}

func isHashedGatewayToken(value string) bool {
	return strings.HasPrefix(value, tokenHashPrefix)
}

func isMaskedCredential(value string) bool {
	return strings.TrimSpace(value) == maskedCredential
}

func requireCredential(value, field string) error {
	if strings.TrimSpace(value) == "" || isMaskedCredential(value) {
		return errors.New(field + "不能为空")
	}
	return nil
}

func maskPageRecords[T any](result response.PageResult, mask func(*T)) response.PageResult {
	records, ok := result.Records.([]T)
	if !ok {
		return result
	}
	for index := range records {
		mask(&records[index])
	}
	result.Records = records
	return result
}

func redactSensitiveText(value string, secrets ...string) string {
	redacted := sensitiveHeaderPattern.ReplaceAllStringFunc(value, func(match string) string {
		for _, separator := range []string{"Bearer ", "bearer ", ":", "="} {
			if index := strings.Index(match, separator); index >= 0 {
				return match[:index+len(separator)] + maskedCredential
			}
		}
		return maskedCredential
	})
	for _, secret := range secrets {
		secret = strings.TrimSpace(secret)
		if secret != "" {
			redacted = strings.ReplaceAll(redacted, secret, maskedCredential)
		}
	}
	return redacted
}
