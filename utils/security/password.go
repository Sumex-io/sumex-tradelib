package security

import (
	"crypto/sha256"
	"fmt"
	"sync/atomic"
)

var passwordSalt atomic.Value // string

// SetPasswordSalt инициализирует глобальную соль (pepper).
func SetPasswordSalt(salt string) {
	passwordSalt.Store(salt)
}

// HashPassword возвращает sha256(password + salt).
func HashPassword(password string) (string, error) {
	salt, _ := passwordSalt.Load().(string)
	h := sha256.New()
	if _, err := h.Write([]byte(password + salt)); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
