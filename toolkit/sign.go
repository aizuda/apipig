package toolkit

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"hash"
)

func HmacMd5(secretKey, message string) string {
	return HmacHash(md5.New, secretKey, message)
}

func HmacSha1(secretKey, message string) string {
	return HmacHash(sha1.New, secretKey, message)
}

// HmacHash 签名 hash 哈希函数，secretKey 签名密钥，message 签名内容
func HmacHash(hash func() hash.Hash, secretKey, message string) string {
	h := hmac.New(hash, []byte(secretKey))
	h.Write([]byte(message))
	signature := h.Sum(nil)
	return hex.EncodeToString(signature)
}
