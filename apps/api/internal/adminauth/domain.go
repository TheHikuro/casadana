package adminauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type AdminUser struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already registered")
	ErrNotFound           = errors.New("admin user not found")
	ErrCannotDeleteSelf   = errors.New("cannot delete your own account")
	ErrInvalidToken       = errors.New("invalid or expired session")
)

const sessionTTL = 12 * time.Hour

func hashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("adminauth: hash password: %w", err)
	}
	return string(b), nil
}

func verifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// signToken produces "<adminID>.<expiryUnix>.<hexHMAC>". adminID is a UUID
// (never contains '.'), so '.' is a safe delimiter.
func signToken(adminID string, secret string, now time.Time) string {
	exp := now.Add(sessionTTL).Unix()
	payload := fmt.Sprintf("%s.%d", adminID, exp)
	sig := hex.EncodeToString(hmacSum(payload, secret))
	return payload + "." + sig
}

func verifyToken(token, secret string, now time.Time) (adminID string, err error) {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return "", ErrInvalidToken
	}
	adminID, expStr, sig := parts[0], parts[1], parts[2]
	payload := adminID + "." + expStr
	expected := hex.EncodeToString(hmacSum(payload, secret))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return "", ErrInvalidToken
	}
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || now.Unix() > exp {
		return "", ErrInvalidToken
	}
	return adminID, nil
}

func hmacSum(payload, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return mac.Sum(nil)
}
