package toolkit

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
)

// GenerateRSAKeyPair generates an RSA key pair. The private key is encoded as
// PKCS#1 DER and the public key is encoded as a Base64 PKIX public key.
func GenerateRSAKeyPair(bits int) ([]byte, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, "", fmt.Errorf("generate RSA key pair: %w", err)
	}

	publicKey, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, "", fmt.Errorf("marshal RSA public key: %w", err)
	}

	return x509.MarshalPKCS1PrivateKey(privateKey), base64.StdEncoding.EncodeToString(publicKey), nil
}

// ParseRSAPrivateKey parses a PKCS#1 DER encoded RSA private key.
func ParseRSAPrivateKey(privateKey []byte) (*rsa.PrivateKey, error) {
	key, err := x509.ParsePKCS1PrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("parse RSA private key: %w", err)
	}
	return key, nil
}

// DecryptRSAOAEP decrypts a Base64 encoded ciphertext using RSA-OAEP SHA-256.
func DecryptRSAOAEP(privateKey *rsa.PrivateKey, ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode RSA ciphertext: %w", err)
	}

	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, data, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt RSA ciphertext: %w", err)
	}
	return string(plaintext), nil
}
