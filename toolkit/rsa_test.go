package toolkit

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRSAKeyPairOAEP(t *testing.T) {
	privateKeyDER, publicKeyBase64, err := GenerateRSAKeyPair(2048)
	assert.NoError(t, err)

	publicKeyDER, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	assert.NoError(t, err)
	parsedPublicKey, err := x509.ParsePKIXPublicKey(publicKeyDER)
	assert.NoError(t, err)
	publicKey, ok := parsedPublicKey.(*rsa.PublicKey)
	assert.True(t, ok)

	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, []byte("admin"), nil)
	assert.NoError(t, err)
	privateKey, err := ParseRSAPrivateKey(privateKeyDER)
	assert.NoError(t, err)
	plaintext, err := DecryptRSAOAEP(privateKey, base64.StdEncoding.EncodeToString(ciphertext))
	assert.NoError(t, err)
	assert.Equal(t, "admin", plaintext)
}
