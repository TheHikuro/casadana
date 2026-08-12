package booking

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type Service struct {
	repo   Repository
	mailer Mailer
	allow  VillaAllowlist
	clock  Clock
}

func NewService(repo Repository, mailer Mailer, allow VillaAllowlist, clock Clock) *Service {
	return &Service{repo: repo, mailer: mailer, allow: allow, clock: clock}
}

type CreateCommand struct {
	VillaSlug  string
	GuestName  string
	GuestEmail string
	GuestPhone string
	CheckIn    time.Time
	CheckOut   time.Time
	Adults     int
	Children   int
	Message    string
	Source     string
	Locale     string
}

func (s *Service) Create(ctx context.Context, cmd CreateCommand) (*Booking, error) {
	if !s.allow.IsKnown(cmd.VillaSlug) {
		return nil, ErrUnknownVilla
	}

	overlapping, err := s.repo.FindOverlapping(ctx, cmd.VillaSlug, cmd.CheckIn, cmd.CheckOut)
	if err != nil {
		return nil, fmt.Errorf("booking: check overlap: %w", err)
	}
	if len(overlapping) > 0 {
		return nil, ErrDatesConflict
	}

	b, err := NewBooking(NewBookingInput{
		VillaSlug:  cmd.VillaSlug,
		GuestName:  cmd.GuestName,
		GuestEmail: cmd.GuestEmail,
		GuestPhone: cmd.GuestPhone,
		CheckIn:    cmd.CheckIn,
		CheckOut:   cmd.CheckOut,
		Adults:     cmd.Adults,
		Children:   cmd.Children,
		Message:    cmd.Message,
		Source:     cmd.Source,
		Locale:     cmd.Locale,
		Now:        s.clock.Now(),
	})
	if err != nil {
		return nil, err
	}

	if err := s.repo.Save(ctx, b); err != nil {
		return nil, fmt.Errorf("booking: save: %w", err)
	}

	// Best-effort emails: a transient mail failure must not lose the booking.
	if err := s.mailer.SendRequestReceived(ctx, b); err != nil {
		slog.WarnContext(ctx, "guest request-received email failed", "booking_id", b.ID, "err", err.Error())
	}
	if err := s.mailer.SendOwnerNewRequest(ctx, b); err != nil {
		slog.WarnContext(ctx, "owner new-request email failed", "booking_id", b.ID, "err", err.Error())
	}

	return b, nil
}

func (s *Service) Availability(ctx context.Context, villaSlug string, from, to time.Time) (Availability, error) {
	if !s.allow.IsKnown(villaSlug) {
		return Availability{}, ErrUnknownVilla
	}
	if !to.After(from) {
		return Availability{}, fmt.Errorf("booking: 'to' must be after 'from'")
	}
	booked, err := s.repo.BookedRanges(ctx, villaSlug, from, to)
	if err != nil {
		return Availability{}, fmt.Errorf("booking: booked ranges: %w", err)
	}
	pending, err := s.repo.PendingRanges(ctx, villaSlug, from, to)
	if err != nil {
		return Availability{}, fmt.Errorf("booking: pending ranges: %w", err)
	}
	return Availability{Booked: booked, Pending: pending}, nil
}

// Get returns a booking by id. Returns ErrNotFound if missing.
func (s *Service) Get(ctx context.Context, id string) (*Booking, error) {
	return s.repo.Get(ctx, id)
}

// List returns a page of bookings ordered by created_at DESC, with optional
// villa_slug and status filters. page is 1-based, limit is clamped to
// [1, 100], default 20.
func (s *Service) List(ctx context.Context, villaSlug *string, status *Status, page, limit int) ([]Booking, int, error) {
	if villaSlug != nil && !s.allow.IsKnown(*villaSlug) {
		return nil, 0, ErrUnknownVilla
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	bookings, err := s.repo.List(ctx, villaSlug, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("booking: list: %w", err)
	}
	total, err := s.repo.Count(ctx, villaSlug, status)
	if err != nil {
		return nil, 0, fmt.Errorf("booking: count: %w", err)
	}
	return bookings, total, nil
}

// Delete hard-deletes a booking. Returns ErrNotFound if no row matched.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// TransitionStatus moves a booking through its lifecycle (pending → approved /
// rejected / cancelled / paid). The state machine is enforced by the domain
// helper Booking.Transition; the service handles persistence.
//
// Approving a booking additionally re-checks for date conflicts against other
// already-confirmed (approved/paid) bookings. This guards against
// double-booking a range that only became blocked after this booking was
// created (e.g. two pending requests racing each other) — creation-time
// conflict checking alone can't catch that, since neither request conflicted
// with anything confirmed yet at the time it was submitted. Once a booking is
// already approved, transitioning it further (approved -> paid) never needs
// this check again: nothing else could have validly become confirmed for an
// overlapping range in the meantime, since that transition would have been
// blocked by this same guard.
func (s *Service) TransitionStatus(ctx context.Context, id string, next Status) (*Booking, error) {
	current, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	transitioned, err := current.Transition(next, s.clock.Now())
	if err != nil {
		return nil, err
	}
	if next == StatusApproved {
		overlapping, err := s.repo.FindOverlappingConfirmed(ctx, current.VillaSlug, current.CheckIn, current.CheckOut, current.ID)
		if err != nil {
			return nil, fmt.Errorf("booking: check overlap: %w", err)
		}
		if len(overlapping) > 0 {
			return nil, ErrDatesConflict
		}
	}
	if err := s.repo.UpdateStatus(ctx, id, transitioned.Status, transitioned.UpdatedAt); err != nil {
		return nil, fmt.Errorf("booking: update status: %w", err)
	}
	s.notifyTransition(ctx, &transitioned)
	return &transitioned, nil
}

// notifyTransition tells the guest what just happened to their request. Like
// creation-time mail this is best-effort: the transition is already persisted,
// and failing the request here would leave the caller thinking the status did
// not change when it did.
//
// StatusPaid sends nothing: payment is recorded by the owners after the fact,
// and the guest already got the confirmation when the booking was approved.
func (s *Service) notifyTransition(ctx context.Context, b *Booking) {
	var (
		err  error
		kind string
	)
	switch b.Status {
	case StatusApproved:
		kind, err = "approved", s.mailer.SendApproved(ctx, b)
	case StatusRejected:
		kind, err = "rejected", s.mailer.SendRejected(ctx, b)
	case StatusCancelled:
		kind, err = "cancelled", s.mailer.SendCancelled(ctx, b)
	default:
		return
	}
	if err != nil {
		slog.WarnContext(ctx, "booking status email failed",
			"booking_id", b.ID, "status", string(b.Status), "kind", kind, "err", err.Error())
	}
}
