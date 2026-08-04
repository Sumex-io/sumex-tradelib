package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
)

type aesKeyState struct {
	key   []byte
	ready bool
}

var gAES atomic.Value // aesKeyState

// SetAESKeyRaw задаёт ключ ровно 16/24/32 байта (AES-128/192/256).
func SetAESKeyRaw(key []byte) error {
	l := len(key)
	if l != 16 && l != 24 && l != 32 {
		return fmt.Errorf("aes: key length must be 16, 24 or 32 bytes")
	}
	gAES.Store(aesKeyState{key: append([]byte(nil), key...), ready: true})
	return nil
}

// SetAESKeyFromPassphrase принимает любую строку и производит из неё 32-байтный ключ (sha256).
func SetAESKeyFromPassphrase(pass string) error {
	sum := sha256.Sum256([]byte(pass))
	gAES.Store(aesKeyState{key: sum[:], ready: true})
	return nil
}

// EncryptString шифрует строку AES-GCM, формат: "v1:" + base64(nonce||ciphertext).
func EncryptString(plaintext string) (string, error) {
	st, _ := gAES.Load().(aesKeyState)
	if !st.ready {
		return "", fmt.Errorf("aes: key not set")
	}
	block, err := aes.NewCipher(st.key)
	if err != nil {
		return "", fmt.Errorf("cipher creation error: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("GCM creation error: %v", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce generation error: %v", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return "v1:" + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString расшифровывает строку формата "v1:"+base64(nonce||ciphertext).
func DecryptString(ciphertext string) (string, error) {
	st, _ := gAES.Load().(aesKeyState)
	if !st.ready {
		return "", fmt.Errorf("aes: key not set")
	}
	if strings.HasPrefix(ciphertext, "v1:") {
		ciphertext = strings.TrimPrefix(ciphertext, "v1:")
	}
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("base64 decoding error: %v", err)
	}
	block, err := aes.NewCipher(st.key)
	if err != nil {
		return "", fmt.Errorf("cipher creation error: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("GCM creation error: %v", err)
	}
	n := gcm.NonceSize()
	if len(data) < n {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, enc := data[:n], data[n:]
	plain, err := gcm.Open(nil, nonce, enc, nil)
	if err != nil {
		return "", fmt.Errorf("decryption error: %v", err)
	}
	return string(plain), nil
}
