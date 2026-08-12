package booking

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusApproved  Status = "approved"
	StatusRejected  Status = "rejected"
	StatusCancelled Status = "cancelled"
	StatusPaid      Status = "paid"
)

type Booking struct {
	ID         string
	VillaSlug  string
	GuestName  string
	GuestEmail string
	GuestPhone string
	CheckIn    time.Time
	CheckOut   time.Time
	Adults     int
	Children   int
	Message    string
	Status     Status
	Source     string
	// Locale the guest browsed the site in ("fr", "en" or "es"). Persisted so
	// mail sent later in the lifecycle — approval, refusal, cancellation — is
	// still written in the guest's language.
	Locale    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type NewBookingInput struct {
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
	Now        time.Time // injected so tests are deterministic
}

// supportedLocales mirrors both the website's message files and the
// bookings_locale_valid CHECK constraint. Anything else — a missing locale, a
// regional tag, a language the site does not ship — falls back to the source
// language rather than being rejected: a booking must never be lost over the
// language its confirmation will be written in.
var supportedLocales = map[string]bool{"fr": true, "en": true, "es": true}

const defaultLocale = "fr"

func normalizeLocale(s string) string {
	base, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(s)), "-")
	if supportedLocales[base] {
		return base
	}
	return defaultLocale
}

var (
	ErrDatesConflict = errors.New("those dates are not available")
	ErrUnknownVilla  = errors.New("unknown villa")
	ErrInvalidStatus = errors.New("invalid booking status transition")
	ErrNotFound      = errors.New("booking not found")
)

func NewBooking(in NewBookingInput) (*Booking, error) {
	in.GuestName = strings.TrimSpace(in.GuestName)
	in.GuestEmail = strings.TrimSpace(in.GuestEmail)
	in.VillaSlug = strings.TrimSpace(in.VillaSlug)
	in.Source = strings.TrimSpace(in.Source)
	if in.Source == "" {
		in.Source = "direct"
	}
	in.Locale = normalizeLocale(in.Locale)

	if in.VillaSlug == "" {
		return nil, errors.New("villa_slug required")
	}
	if in.GuestName == "" {
		return nil, errors.New("guest_name required")
	}
	if in.GuestEmail == "" || !strings.Contains(in.GuestEmail, "@") {
		return nil, errors.New("guest_email invalid")
	}
	if in.Adults < 1 {
		return nil, errors.New("adults must be >= 1")
	}
	if in.Children < 0 {
		return nil, errors.New("children must be >= 0")
	}
	if !in.CheckOut.After(in.CheckIn) {
		return nil, errors.New("check_out must be after check_in")
	}
	if in.CheckIn.Before(in.Now.Truncate(24 * time.Hour)) {
		return nil, errors.New("check_in must not be in the past")
	}

	now := in.Now
	return &Booking{
		ID:         uuid.NewString(),
		VillaSlug:  in.VillaSlug,
		GuestName:  in.GuestName,
		GuestEmail: in.GuestEmail,
		GuestPhone: in.GuestPhone,
		CheckIn:    in.CheckIn,
		CheckOut:   in.CheckOut,
		Adults:     in.Adults,
		Children:   in.Children,
		Message:    in.Message,
		Status:     StatusPending,
		Source:     in.Source,
		Locale:     in.Locale,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// Transition returns the booking with a new status if the transition is allowed.
func (b Booking) Transition(next Status, now time.Time) (Booking, error) {
	allowed := map[Status]map[Status]bool{
		StatusPending:  {StatusApproved: true, StatusRejected: true, StatusCancelled: true},
		StatusApproved: {StatusPaid: true, StatusCancelled: true},
		StatusPaid:     {StatusCancelled: true},
	}
	if !allowed[b.Status][next] {
		return Booking{}, ErrInvalidStatus
	}
	b.Status = next
	b.UpdatedAt = now
	return b, nil
}
