package adminauth

import (
	"context"
	"time"
)

type Repository interface {
	Save(ctx context.Context, u *AdminUser) error                      // ErrEmailTaken on unique conflict
	FindByEmail(ctx context.Context, email string) (*AdminUser, error) // ErrNotFound
	FindByID(ctx context.Context, id string) (*AdminUser, error)       // ErrNotFound
	List(ctx context.Context) ([]AdminUser, error)
	Delete(ctx context.Context, id string) error // ErrNotFound if 0 rows affected
}

type Clock interface {
	Now() time.Time
}
