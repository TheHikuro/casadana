package booking

import (
	"context"
	"time"
)

type Repository interface {
	Save(ctx context.Context, b *Booking) error
	FindOverlapping(ctx context.Context, villaSlug string, from, to time.Time) ([]Booking, error)
	// FindOverlappingConfirmed is like FindOverlapping but only considers
	// approved/paid bookings (not pending) and excludes excludeID — used to
	// guard the pending->approved transition against double-booking a date
	// range that another booking already confirmed in the meantime.
	FindOverlappingConfirmed(ctx context.Context, villaSlug string, from, to time.Time, excludeID string) ([]Booking, error)
	BookedRanges(ctx context.Context, villaSlug string, from, to time.Time) ([]DateRange, error)
	PendingRanges(ctx context.Context, villaSlug string, from, to time.Time) ([]DateRange, error)
	Get(ctx context.Context, id string) (*Booking, error)
	UpdateStatus(ctx context.Context, id string, status Status, updatedAt time.Time) error
	List(ctx context.Context, villaSlug *string, status *Status, limit, offset int) ([]Booking, error)
	Count(ctx context.Context, villaSlug *string, status *Status) (int, error)
	Delete(ctx context.Context, id string) error
}

// Mailer sends the transactional mail a booking's lifecycle produces. Every
// method is best-effort at the call site: a mail failure never rolls back the
// state change that triggered it.
type Mailer interface {
	// SendRequestReceived acknowledges a new request to the guest. Deliberately
	// not called a confirmation: at this point nothing is confirmed, which is
	// exactly what that email has to say.
	SendRequestReceived(ctx context.Context, b *Booking) error
	// SendOwnerNewRequest tells the owners a request is waiting for them.
	SendOwnerNewRequest(ctx context.Context, b *Booking) error
	// SendApproved tells the guest the stay is confirmed.
	SendApproved(ctx context.Context, b *Booking) error
	// SendRejected tells the guest the request was not accepted, so nobody is
	// left waiting on an answer that never comes.
	SendRejected(ctx context.Context, b *Booking) error
	// SendCancelled confirms a cancellation to the guest.
	SendCancelled(ctx context.Context, b *Booking) error
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
