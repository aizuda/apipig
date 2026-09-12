package toolkit

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

type AES struct {
	Key []byte // 16 bytes for AES-128
}

func pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

func unPad(data []byte, blockSizes ...int) ([]byte, error) {
	blockSize := aes.BlockSize
	if len(blockSizes) > 0 {
		blockSize = blockSizes[0]
	}
	if len(data) == 0 || blockSize <= 0 || len(data)%blockSize != 0 {
		return nil, fmt.Errorf("invalid padded data")
	}
	padding := data[len(data)-1]
	if padding == 0 || int(padding) > blockSize || int(padding) > len(data) {
		return nil, fmt.Errorf("padding size error")
	}
	for _, value := range data[len(data)-int(padding):] {
		if value != padding {
			return nil, fmt.Errorf("padding content error")
		}
	}
	return data[:len(data)-int(padding)], nil
}
func (a *AES) EncryptString(plaintext string) (string, error) {
	encrypt, err := a.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(encrypt), nil
}

// Encrypt data using AES CBC
func (a *AES) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(a.Key)
	if err != nil {
		return nil, err
	}

	plaintext = pad(plaintext, aes.BlockSize)
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], plaintext)

	return ciphertext, nil
}

func (a *AES) DecryptString(ciphertext string) (string, error) {
	decryptBytes, err := a.DecryptBytes(ciphertext)
	return string(decryptBytes), err
}

func (a *AES) DecryptBytes(ciphertext string) ([]byte, error) {
	decodeString, err := hex.DecodeString(ciphertext)
	if err == nil {
		decrypt, err := a.Decrypt(decodeString)
		if err == nil {
			return decrypt, nil
		}
	}
	return nil, errors.New("aes cbc decrypt error")
}

// Decrypt data using AES CBC
func (a *AES) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(a.Key)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext is not a multiple of the block size")
	}
	// CBC 解密会原地写入，复制密文避免修改调用方持有的缓冲区。
	plaintext := make([]byte, len(ciphertext))
	copy(plaintext, ciphertext)

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, plaintext)

	return unPad(plaintext, aes.BlockSize)
}
