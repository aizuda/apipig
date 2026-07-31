package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

const encryptedCredentialPrefix = "enc:v1:"

// CredentialVault 负责可逆上游凭据的加密和解密。
type CredentialVault interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
	Enabled() bool
}

type aesCredentialVault struct {
	keyProvider func() string
	mu          sync.Mutex
	loadedKey   string
	aead        cipher.AEAD
	loadErr     error
}

func newAESCredentialVault(keyProvider func() string) CredentialVault {
	return &aesCredentialVault{keyProvider: keyProvider}
}

func (v *aesCredentialVault) Enabled() bool {
	return strings.TrimSpace(v.keyProvider()) != ""
}

func (v *aesCredentialVault) Encrypt(plaintext string) (string, error) {
	if plaintext == "" || strings.HasPrefix(plaintext, encryptedCredentialPrefix) {
		return plaintext, nil
	}
	if !v.Enabled() {
		return "", errors.New("保存 AI 凭据前必须配置 APIPIG_AI_ENCRYPTION_KEY")
	}
	aead, err := v.loadAEAD()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return encryptedCredentialPrefix + base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (v *aesCredentialVault) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" || !strings.HasPrefix(ciphertext, encryptedCredentialPrefix) {
		return ciphertext, nil
	}
	if !v.Enabled() {
		return "", errors.New("AI 凭据已加密，但未配置 APIPIG_AI_ENCRYPTION_KEY")
	}
	aead, err := v.loadAEAD()
	if err != nil {
		return "", err
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(ciphertext, encryptedCredentialPrefix))
	if err != nil || len(payload) < aead.NonceSize() {
		return "", errors.New("AI 加密凭据格式无效")
	}
	nonce, encrypted := payload[:aead.NonceSize()], payload[aead.NonceSize():]
	plaintext, err := aead.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", errors.New("AI 凭据解密失败，请检查主密钥")
	}
	return string(plaintext), nil
}

func (v *aesCredentialVault) loadAEAD() (cipher.AEAD, error) {
	key := strings.TrimSpace(v.keyProvider())
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.aead != nil {
		return v.aead, nil
	}
	if key == v.loadedKey && (v.aead != nil || v.loadErr != nil) {
		return v.aead, v.loadErr
	}
	v.loadedKey = key
	v.aead = nil
	v.loadErr = nil
	if len(key) < 32 {
		v.loadErr = fmt.Errorf("APIPIG_AI_ENCRYPTION_KEY 长度不能少于 32 个字符")
		return nil, v.loadErr
	}
	derived := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(derived[:])
	if err != nil {
		v.loadErr = err
		return nil, err
	}
	v.aead, v.loadErr = cipher.NewGCM(block)
	return v.aead, v.loadErr
}

func isEncryptedCredential(value string) bool {
	return strings.HasPrefix(value, encryptedCredentialPrefix)
}
