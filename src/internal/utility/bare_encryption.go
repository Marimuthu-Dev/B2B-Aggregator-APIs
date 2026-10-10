package utility

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
	"os"
	"strings"
)

func BaseEncryptUrlSafe(plainText string) string {
	if plainText == "" {
		return ""
	}

	encryptionKey := os.Getenv("ENCRYPTION_KEY")
	if len(encryptionKey) < 32 {
		return "" // Requires at least a 32-byte key
	}
	key := []byte(encryptionKey[:32])
	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}

	pt := []byte(plainText)
	padding := aes.BlockSize - len(pt)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	pt = append(pt, padtext...)

	cipherText := make([]byte, aes.BlockSize+len(pt))
	iv := cipherText[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return ""
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(cipherText[aes.BlockSize:], pt)

	b64 := base64.StdEncoding.EncodeToString(cipherText)
	b64 = strings.ReplaceAll(b64, "+", "-")
	b64 = strings.ReplaceAll(b64, "/", "_")
	b64 = strings.ReplaceAll(b64, "=", "")
	return b64
}

func BaseDecryptUrlSafe(cipherText string) string {
	if cipherText == "" {
		return ""
	}

	encryptionKey := os.Getenv("ENCRYPTION_KEY")
	if len(encryptionKey) < 32 {
		return "" // Requires at least a 32-byte key
	}
	key := []byte(encryptionKey[:32])

	cipherText = strings.ReplaceAll(cipherText, "-", "+")
	cipherText = strings.ReplaceAll(cipherText, "_", "/")
	// Add padding back if necessary
	if m := len(cipherText) % 4; m != 0 {
		cipherText += strings.Repeat("=", 4-m)
	}

	decoded, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil || len(decoded) < aes.BlockSize {
		return ""
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}

	iv := decoded[:aes.BlockSize]
	msg := decoded[aes.BlockSize:]

	if len(msg)%aes.BlockSize != 0 {
		return ""
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(msg, msg)

	// Remove padding
	if len(msg) > 0 {
		paddingLen := int(msg[len(msg)-1])
		if paddingLen > 0 && paddingLen <= aes.BlockSize && paddingLen <= len(msg) {
			msg = msg[:len(msg)-paddingLen]
		}
	}

	return string(msg)
}
