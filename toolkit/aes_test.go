package toolkit

import (
	"encoding/hex"
	"fmt"
	"testing"
)

func TestAES(t *testing.T) {
	aes := &AES{Key: []byte("18ffc810652aa83a")}

	encrypted, err := aes.Encrypt([]byte("01898"))
	if err != nil {
		fmt.Println("Encryption error:", err)
		return
	}

    fmt.Printf("Encrypted: %s\n", hex.EncodeToString(encrypted))
	fmt.Println(len(encrypted))

	decodeString, _ := hex.DecodeString(string(encrypted))
	decrypted, err := aes.Decrypt(decodeString)
	if err != nil {
		fmt.Println("Decryption error:", err)
		return
	}

	fmt.Printf("Decrypted: %s\n", decrypted)
}
