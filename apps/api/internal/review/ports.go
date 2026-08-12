package review

import (
	"context"
	"time"
)

type Repository interface {
	Save(ctx context.Context, r *Review) error
	// ListByVillaAndStatus returns a villa's reviews, featured first then
	// newest first. A nil status means "every status".
	ListByVillaAndStatus(ctx context.Context, slug string, status *Status) ([]Review, error)
	Get(ctx context.Context, id string) (*Review, error)
	Update(ctx context.Context, id string, patch UpdatePatch) (*Review, error)
	Delete(ctx context.Context, id string) error
	// GetMeta returns zero values (not an error) for a villa with no meta row.
	GetMeta(ctx context.Context, slug string) (ReviewMeta, error)
	UpsertMeta(ctx context.Context, m ReviewMeta) (ReviewMeta, error)
}

type BookingReader interface {
	GetVillaSlug(ctx context.Context, bookingID string) (string, error)
}

type Clock interface {
	Now() time.Time
}

// EventRecorder is our own view of the activity log: a moderation action is
// worth a line in the villa's timeline. Defined here rather than imported so
// the slice stays independent of whoever writes those events. A nil recorder
// disables recording, and a recorder error never fails the mutation.
type EventRecorder interface {
	Record(ctx context.Context, villaSlug, message string) error
}
