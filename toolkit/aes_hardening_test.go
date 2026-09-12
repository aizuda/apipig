package toolkit

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAESDecryptRejectsMalformedCiphertext(t *testing.T) {
	aes := &AES{Key: []byte("0123456789abcdef")}
	require.NotPanics(t, func() {
		_, err := aes.Decrypt([]byte{})
		require.Error(t, err)
	})
	require.NotPanics(t, func() {
		_, err := aes.Decrypt(make([]byte, 17))
		require.Error(t, err)
	})
}

func TestAESRejectsInvalidPadding(t *testing.T) {
	_, err := unPad(make([]byte, 16), 16)
	require.Error(t, err)

	data := make([]byte, 16)
	data[15] = 2
	_, err = unPad(data, 16)
	require.Error(t, err)
}

func TestAESRoundTripPreservesEmptyPlaintext(t *testing.T) {
	aes := &AES{Key: []byte("0123456789abcdef")}
	ciphertext, err := aes.Encrypt(nil)
	require.NoError(t, err)
	plaintext, err := aes.Decrypt(ciphertext)
	require.NoError(t, err)
	require.Empty(t, plaintext)
}
