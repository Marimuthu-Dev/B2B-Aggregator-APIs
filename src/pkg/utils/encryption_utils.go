package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"golang.org/x/crypto/pbkdf2"
	"os"

	"golang.org/x/text/encoding/unicode"
)

func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS7Unpadding(plantText []byte) []byte {
	length := len(plantText)
	unpadding := int(plantText[length-1])
	return plantText[:(length - unpadding)]
}

func Encrypt(text string) (string, error) {
	keyString := os.Getenv("LOGIN_ENC_KEY")
	saltString := os.Getenv("LOGIN_ENC_SALT")

	if keyString == "" || saltString == "" {
		return "", errors.New("encryption key/salt not configured")
	}

	// UTF-16LE encoding (matches .NET's Encoding.Unicode)
	enc := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
	inputBytes, err := enc.Bytes([]byte(text))
	if err != nil {
		return "", err
	}

	// PBKDF2 with SHA1, 1000 iterations, 48 bytes (32 key, 16 IV)
	derived := pbkdf2.Key([]byte(keyString), []byte(saltString), 1000, 48, sha1.New)
	key := derived[:32]
	iv := derived[32:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	inputBytes = PKCS7Padding(inputBytes, block.BlockSize())
	blockMode := cipher.NewCBCEncrypter(block, iv)
	encrypted := make([]byte, len(inputBytes))
	blockMode.CryptBlocks(encrypted, inputBytes)

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func Decrypt(cipherTextBase64 string) (string, error) {
	keyString := os.Getenv("LOGIN_ENC_KEY")
	saltString := os.Getenv("LOGIN_ENC_SALT")

	if keyString == "" || saltString == "" {
		return "", errors.New("encryption key/salt not configured")
	}

	cipherBytes, err := base64.StdEncoding.DecodeString(cipherTextBase64)
	if err != nil {
		return "", err
	}

	derived := pbkdf2.Key([]byte(keyString), []byte(saltString), 1000, 48, sha1.New)
	key := derived[:32]
	iv := derived[32:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	if len(cipherBytes)%block.BlockSize() != 0 {
		return "", errors.New("cipherText is not a multiple of the block size")
	}

	blockMode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(cipherBytes))
	blockMode.CryptBlocks(decrypted, cipherBytes)

	decrypted = PKCS7Unpadding(decrypted)

	// UTF-16LE decoding
	dec := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
	result, err := dec.Bytes(decrypted)
	if err != nil {
		return "", err
	}

	return string(result), nil
}
