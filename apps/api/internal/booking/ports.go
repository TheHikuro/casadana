package booking

import (
	"context"
	"time"
)

type Repository interface {
	Save(ctx context.Context, b *Booking) error
	FindOverlapping(ctx context.Context, villaSlug string, from, to time.Time) ([]Booking, error)
	BookedRanges(ctx context.Context, villaSlug string, from, to time.Time) ([]DateRange, error)
	PendingRanges(ctx context.Context, villaSlug string, from, to time.Time) ([]DateRange, error)
	Get(ctx context.Context, id string) (*Booking, error)
	UpdateStatus(ctx context.Context, id string, status Status, updatedAt time.Time) error
	List(ctx context.Context, villaSlug *string, status *Status, limit, offset int) ([]Booking, error)
	Count(ctx context.Context, villaSlug *string, status *Status) (int, error)
	Delete(ctx context.Context, id string) error
}

type Mailer interface {
	SendBookingConfirmation(ctx context.Context, b *Booking) error
	SendAdminNotification(ctx context.Context, b *Booking) error
}

type Clock interface {
	Now() time.Time
}

type VillaAllowlist interface {
	IsKnown(slug string) bool
}

type DateRange struct {
	CheckIn  time.Time
	CheckOut time.Time
}

// Availability separates ranges already confirmed (approved/paid, hard
// blocked) from ranges still pending confirmation (provisionally held: a new
// booking request would conflict, but nothing is guaranteed yet).
type Availability struct {
	Booked  []DateRange
	Pending []DateRange
}
