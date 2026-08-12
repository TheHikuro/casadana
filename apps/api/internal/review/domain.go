package review

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
)

func (s Status) valid() bool {
	switch s {
	case StatusPending, StatusApproved, StatusRejected:
		return true
	default:
		return false
	}
}

// CategoryRatings are the optional per-category scores a review may carry
// alongside its overall rating. A nil field means that category was not scored,
// which keeps it out of the villa's average for that category instead of
// dragging it down with a zero. Each score is 1..5.
type CategoryRatings struct {
	Cleanliness *float64
	Comfort     *float64
	Location    *float64
	Host        *float64
	Value       *float64
}

// Review is a guest- or admin-authored testimonial for a villa. BookingID is
// empty for admin-authored reviews: those are transcribed from Airbnb /
// Booking.com and have no booking row of ours behind them.
type Review struct {
	ID         string
	BookingID  string
	VillaSlug  string
	AuthorName string
	Rating     int
	Body       string
	Status     Status
	Meta       string
	Source     string
	Featured   bool
	Categories CategoryRatings
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type NewReviewInput struct {
	BookingID  string
	VillaSlug  string
	AuthorName string
	Rating     int
	Body       string
	Now        time.Time
}

// NewAdminReviewInput builds a review with no booking behind it. Status
// defaults to approved (an admin transcribing a review has already vetted it).
type NewAdminReviewInput struct {
	VillaSlug  string
	AuthorName string
	Rating     int
	Body       string
	Status     Status
	Meta       string
	Source     string
	Featured   bool
	Categories CategoryRatings
	Now        time.Time
}

// UpdatePatch carries a partial moderation edit. A nil field means "leave
// unchanged" — including each individual category score.
type UpdatePatch struct {
	Status     *Status
	Featured   *bool
	Meta       *string
	Source     *string
	Body       *string
	Rating     *int
	Categories CategoryRatings
}

// Breakdown holds the five per-category averages shown next to the overall
// rating. A nil score means no approved review has rated that category yet, so
// the bar is left off rather than drawn at zero.
type Breakdown struct {
	Cleanliness *float64
	Comfort     *float64
	Location    *float64
	Host        *float64
	Value       *float64
}

// ReviewMeta is the villa's published rating, computed from its approved
// reviews. Nothing in it is stored or hand-entered: moderating a review into or
// out of `approved` is what moves these numbers, so what a guest reads always
// matches the reviews on show.
type ReviewMeta struct {
	VillaSlug    string
	DisplayAvg   float64
	DisplayCount int
	Breakdown    Breakdown
}

var (
	ErrBookingNotFound = errors.New("booking not found")
	ErrAlreadyReviewed = errors.New("review already exists for this booking")
	ErrNotFound        = errors.New("review not found")
	ErrInvalidPayload  = errors.New("invalid review payload")
)

func NewReview(in NewReviewInput) (*Review, error) {
	in.AuthorName = strings.TrimSpace(in.AuthorName)
	in.Body = strings.TrimSpace(in.Body)

	if in.BookingID == "" {
		return nil, ErrInvalidPayload
	}
	if in.VillaSlug == "" {
		return nil, ErrInvalidPayload
	}
	if err := validateContent(in.AuthorName, in.Rating, in.Body); err != nil {
		return nil, err
	}

	now := in.Now
	return &Review{
		ID:         uuid.NewString(),
		BookingID:  in.BookingID,
		VillaSlug:  in.VillaSlug,
		AuthorName: in.AuthorName,
		Rating:     in.Rating,
		Body:       in.Body,
		Status:     StatusPending,
		Source:     "direct",
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func NewAdminReview(in NewAdminReviewInput) (*Review, error) {
	in.AuthorName = strings.TrimSpace(in.AuthorName)
	in.Body = strings.TrimSpace(in.Body)

	if in.VillaSlug == "" {
		return nil, ErrInvalidPayload
	}
	if err := validateContent(in.AuthorName, in.Rating, in.Body); err != nil {
		return nil, err
	}
	if in.Status == "" {
		in.Status = StatusApproved
	}
	if !in.Status.valid() {
		return nil, ErrInvalidPayload
	}
	if len(in.Meta) > 2000 || len(in.Source) > 64 {
		return nil, ErrInvalidPayload
	}
	if err := in.Categories.Validate(); err != nil {
		return nil, err
	}

	now := in.Now
	return &Review{
		ID:         uuid.NewString(),
		VillaSlug:  in.VillaSlug,
		AuthorName: in.AuthorName,
		Rating:     in.Rating,
		Body:       in.Body,
		Status:     in.Status,
		Meta:       in.Meta,
		Source:     in.Source,
		Featured:   in.Featured,
		Categories: in.Categories,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// Validate rejects any category score that was set but sits outside 1..5. An
// unset score is always fine: it simply means "not rated".
func (c CategoryRatings) Validate() error {
	for _, s := range []*float64{c.Cleanliness, c.Comfort, c.Location, c.Host, c.Value} {
		if s != nil && (*s < 1 || *s > 5) {
			return ErrInvalidPayload
		}
	}
	return nil
}

// Validate rejects a patch whose set fields are out of range. An entirely
// empty patch is valid: it is a no-op, not an error.
func (p UpdatePatch) Validate() error {
	if p.Status != nil && !p.Status.valid() {
		return ErrInvalidPayload
	}
	if p.Rating != nil && (*p.Rating < 1 || *p.Rating > 5) {
		return ErrInvalidPayload
	}
	if p.Body != nil && len(*p.Body) > 2000 {
		return ErrInvalidPayload
	}
	if p.Meta != nil && len(*p.Meta) > 2000 {
		return ErrInvalidPayload
	}
	if p.Source != nil && len(*p.Source) > 64 {
		return ErrInvalidPayload
	}
	return p.Categories.Validate()
}

func validateContent(authorName string, rating int, body string) error {
	if authorName == "" || len(authorName) > 120 {
		return ErrInvalidPayload
	}
	if rating < 1 || rating > 5 {
		return ErrInvalidPayload
	}
	if len(body) > 2000 {
		return ErrInvalidPayload
	}
	return nil
}
