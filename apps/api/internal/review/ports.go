package review

import (
	"context"
	"time"
)

type Repository interface {
	Save(ctx context.Context, r *Review) error
	ListByVillaSlug(ctx context.Context, slug string) ([]Review, error)
	Delete(ctx context.Context, id string) error
}

type BookingReader interface {
	GetVillaSlug(ctx context.Context, bookingID string) (string, error)
}

type Clock interface {
	Now() time.Time
}
