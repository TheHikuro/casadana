package adminauth

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repo   Repository
	clock  Clock
	secret string
}

func NewService(repo Repository, clock Clock, secret string) *Service {
	return &Service{repo: repo, clock: clock, secret: secret}
}

// Login returns a signed session token and the authenticated user, or
// ErrInvalidCredentials for either an unknown email or a wrong password
// (deliberately not distinguished, to avoid leaking which one was wrong).
func (s *Service) Login(ctx context.Context, email, password string) (string, *AdminUser, error) {
	u, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return "", nil, ErrInvalidCredentials
	}
	if !verifyPassword(u.PasswordHash, password) {
		return "", nil, ErrInvalidCredentials
	}
	token := signToken(u.ID, s.secret, s.clock.Now())
	return token, u, nil
}

// Authenticate validates a session token and returns the current admin user.
// Returns ErrInvalidToken for a malformed/expired/tampered token, or if the
// admin it names was deleted after the token was issued.
func (s *Service) Authenticate(ctx context.Context, token string) (*AdminUser, error) {
	adminID, err := verifyToken(token, s.secret, s.clock.Now())
	if err != nil {
		return nil, ErrInvalidToken
	}
	u, err := s.repo.FindByID(ctx, adminID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	return u, nil
}

func (s *Service) CreateUser(ctx context.Context, email, password string) (*AdminUser, error) {
	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	u := &AdminUser{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    s.clock.Now(),
	}
	if err := s.repo.Save(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Service) ListUsers(ctx context.Context) ([]AdminUser, error) {
	return s.repo.List(ctx)
}

// DeleteUser removes an admin account. Blocking self-delete unconditionally
// (rather than only when you're the last admin) is sufficient on its own to
// guarantee at least one admin always remains — see Task C3's design note.
func (s *Service) DeleteUser(ctx context.Context, callerID, targetID string) error {
	if callerID == targetID {
		return ErrCannotDeleteSelf
	}
	return s.repo.Delete(ctx, targetID)
}
