package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a password with bcrypt.
func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword compares a bcrypt hash with a plain password.
func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// Minimal unsigned token (NOT secure / placeholder) Format: base64(email)|unix_expiry
func GenerateToken(email string, ttl time.Duration) string { // legacy placeholder
	exp := time.Now().Add(ttl).Unix()
	return base64.RawStdEncoding.EncodeToString([]byte(email)) + "|" + fmt.Sprintf("%d", exp)
}

// ParseToken parses the placeholder token.
func ParseToken(tok string) (string, time.Time, error) { // legacy placeholder
	parts := strings.Split(tok, "|")
	if len(parts) != 2 {
		return "", time.Time{}, errors.New("invalid token")
	}
	emailRaw, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return "", time.Time{}, err
	}
	var expUnix int64
	if _, err = fmt.Sscanf(parts[1], "%d", &expUnix); err != nil {
		return "", time.Time{}, err
	}
	return string(emailRaw), time.Unix(expUnix, 0), nil
}

// Lightweight HMAC "JWT-like" token (not full standard) format: b64(email)|b64(role)|expUnix|sig
func GenerateJWT(email, role string, ttl time.Duration, secret string) (string, error) {
	exp := time.Now().Add(ttl).Unix()
	parts := []string{
		base64.RawStdEncoding.EncodeToString([]byte(email)),
		base64.RawStdEncoding.EncodeToString([]byte(role)),
		fmt.Sprintf("%d", exp),
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strings.Join(parts, "|")))
	sig := base64.RawStdEncoding.EncodeToString(mac.Sum(nil))
	parts = append(parts, sig)
	return strings.Join(parts, "|"), nil
}

func ParseJWT(tok, secret string) (email, role string, exp time.Time, err error) {
	parts := strings.Split(tok, "|")
	if len(parts) != 4 {
		return "", "", time.Time{}, errors.New("invalid token")
	}
	emailBytes, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", time.Time{}, err
	}
	roleBytes, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", time.Time{}, err
	}
	var expUnix int64
	if _, err = fmt.Sscanf(parts[2], "%d", &expUnix); err != nil {
		return "", "", time.Time{}, err
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strings.Join(parts[:3], "|")))
	expected := base64.RawStdEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[3])) {
		return "", "", time.Time{}, errors.New("signature")
	}
	return string(emailBytes), string(roleBytes), time.Unix(expUnix, 0), nil
}
