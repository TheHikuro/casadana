package adminauth

import (
	"context"
	"testing"
	"time"
)

func seedUser(t *testing.T, repo *fakeRepo, id, email, password string, now time.Time) AdminUser {
	t.Helper()
	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	u := AdminUser{ID: id, Email: email, PasswordHash: hash, CreatedAt: now}
	repo.users = append(repo.users, u)
	return u
}

func TestLogin_Happy(t *testing.T) {
	repo := &fakeRepo{}
	seedUser(t, repo, "admin-1", "loan@casa-dana.com", "correcthorse", d("2026-07-02"))
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")

	token, admin, err := svc.Login(context.Background(), "loan@casa-dana.com", "correcthorse")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if token == "" {
		t.Error("token was empty")
	}
	if admin.Email != "loan@casa-dana.com" {
		t.Errorf("admin email = %q", admin.Email)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := &fakeRepo{}
	seedUser(t, repo, "admin-1", "loan@casa-dana.com", "correcthorse", d("2026-07-02"))
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")

	_, _, err := svc.Login(context.Background(), "loan@casa-dana.com", "wrong")
	if err != ErrInvalidCredentials {
		t.Fatalf("err = %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	svc := NewService(&fakeRepo{}, fixedClock{t: d("2026-07-02")}, "test-secret")

	_, _, err := svc.Login(context.Background(), "ghost@casa-dana.com", "whatever")
	if err != ErrInvalidCredentials {
		t.Fatalf("err = %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthenticate_ValidToken(t *testing.T) {
	repo := &fakeRepo{}
	seedUser(t, repo, "admin-1", "loan@casa-dana.com", "correcthorse", d("2026-07-02"))
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")

	token, _, err := svc.Login(context.Background(), "loan@casa-dana.com", "correcthorse")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	admin, err := svc.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if admin.Email != "loan@casa-dana.com" {
		t.Errorf("admin email = %q", admin.Email)
	}
}

func TestAuthenticate_InvalidToken(t *testing.T) {
	svc := NewService(&fakeRepo{}, fixedClock{t: d("2026-07-02")}, "test-secret")
	if _, err := svc.Authenticate(context.Background(), "garbage"); err != ErrInvalidToken {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	repo := &fakeRepo{}
	seedUser(t, repo, "admin-1", "loan@casa-dana.com", "correcthorse", d("2026-07-02"))
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")

	_, err := svc.CreateUser(context.Background(), "loan@casa-dana.com", "anotherpassword")
	if err != ErrEmailTaken {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
}

func TestCreateUser_Happy(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")

	admin, err := svc.CreateUser(context.Background(), "new@casa-dana.com", "supersecret1")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if admin.Email != "new@casa-dana.com" {
		t.Errorf("email = %q", admin.Email)
	}
	if len(repo.users) != 1 {
		t.Errorf("repo.users len = %d, want 1", len(repo.users))
	}
}

func TestDeleteUser_CannotDeleteSelf(t *testing.T) {
	repo := &fakeRepo{}
	a := seedUser(t, repo, "admin-1", "loan@casa-dana.com", "pw", d("2026-07-02"))
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")

	if err := svc.DeleteUser(context.Background(), a.ID, a.ID); err != ErrCannotDeleteSelf {
		t.Fatalf("err = %v, want ErrCannotDeleteSelf", err)
	}
}

func TestDeleteUser_Happy(t *testing.T) {
	repo := &fakeRepo{}
	a := seedUser(t, repo, "admin-1", "loan@casa-dana.com", "pw1", d("2026-07-02"))
	b := seedUser(t, repo, "admin-2", "co-host@casa-dana.com", "pw2", d("2026-07-02"))
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")

	if err := svc.DeleteUser(context.Background(), a.ID, b.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	if len(repo.users) != 1 {
		t.Errorf("repo.users len = %d, want 1", len(repo.users))
	}
}
