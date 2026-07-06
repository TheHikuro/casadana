package adminauth

import (
	"testing"
	"time"
)

func d(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestSignVerifyToken_RoundTrip(t *testing.T) {
	now := d("2026-07-02")
	token := signToken("admin-1", "secret", now)
	id, err := verifyToken(token, "secret", now.Add(time.Hour))
	if err != nil {
		t.Fatalf("verifyToken: %v", err)
	}
	if id != "admin-1" {
		t.Errorf("id = %q, want admin-1", id)
	}
}

func TestVerifyToken_Expired(t *testing.T) {
	now := d("2026-07-02")
	token := signToken("admin-1", "secret", now)
	if _, err := verifyToken(token, "secret", now.Add(13*time.Hour)); err != ErrInvalidToken {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestVerifyToken_TamperedSignature(t *testing.T) {
	now := d("2026-07-02")
	token := signToken("admin-1", "secret", now)
	tampered := token[:len(token)-1] + "0"
	if _, err := verifyToken(tampered, "secret", now); err != ErrInvalidToken {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestVerifyToken_WrongSecret(t *testing.T) {
	now := d("2026-07-02")
	token := signToken("admin-1", "secret-a", now)
	if _, err := verifyToken(token, "secret-b", now); err != ErrInvalidToken {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestVerifyToken_Malformed(t *testing.T) {
	if _, err := verifyToken("not-a-token", "secret", d("2026-07-02")); err != ErrInvalidToken {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := hashPassword("correcthorse")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	if !verifyPassword(hash, "correcthorse") {
		t.Error("verifyPassword should accept the correct password")
	}
	if verifyPassword(hash, "wrong") {
		t.Error("verifyPassword should reject the wrong password")
	}
}
