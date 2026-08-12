package audit

import (
	"context"
	"time"
)

type Repository interface {
	// Save appends the event and hydrates e.CreatedAt with the timestamp the
	// database assigned.
	Save(ctx context.Context, e *Event) error
	List(ctx context.Context, villaSlug string, limit, offset int) ([]Event, error)
	Count(ctx context.Context, villaSlug string) (int, error)
}

type Clock interface {
	Now() time.Time
}
