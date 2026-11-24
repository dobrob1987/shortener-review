package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
)

const secretKey = "super-secret-key"

var (
	ErrMalformedCookie  = errors.New("некорректный формат cookie")
	ErrInvalidSignature = errors.New("неверная подпись cookie")
)

func sign(userID string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(userID))

	return hex.EncodeToString(h.Sum(nil))
}

func verify(userID, signature string) bool {
	expectedSignature := sign(userID)

	return hmac.Equal(
		[]byte(expectedSignature),
		[]byte(signature),
	)
}

func NewCookie() *http.Cookie {
	userID := rand.Text()

	return &http.Cookie{
		Name:     "user_id",
		Value:    userID + ":" + sign(userID),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func ParseCookie(value string) (string, error) {
	userID, signature, found := strings.Cut(value, ":")

	if !found || userID == "" || signature == "" ||
		strings.Contains(signature, ":") {
		return "", ErrMalformedCookie
	}

	if !verify(userID, signature) {
		return "", ErrInvalidSignature
	}

	return userID, nil
}

func GetOrCreateUser(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie("user_id")
	if err == nil {
		userID, parseErr := ParseCookie(cookie.Value)
		if parseErr == nil {
			return userID
		}
	}

	cookie = NewCookie()
	http.SetCookie(w, cookie)

	userID, _, _ := strings.Cut(cookie.Value, ":")
	return userID
}
