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
	Now        time.Time
}

// UpdatePatch carries a partial moderation edit. A nil field means "leave
// unchanged".
type UpdatePatch struct {
	Status   *Status
	Featured *bool
	Meta     *string
	Source   *string
	Body     *string
	Rating   *int
}

// Breakdown holds the five per-axis scores shown next to the overall rating.
// Each is 0..5.
type Breakdown struct {
	Cleanliness float64
	Comfort     float64
	Location    float64
	Host        float64
	Value       float64
}

// ReviewMeta is the villa-level display aggregate. It is admin-curated rather
// than computed: the public numbers include reviews left on Airbnb /
// Booking.com that we never store row by row.
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
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
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
	return nil
}

// Validate rejects out-of-range display aggregates. Scores are 0..5; a zero
// score means "not published" rather than "rated zero".
func (m ReviewMeta) Validate() error {
	if m.VillaSlug == "" {
		return ErrInvalidPayload
	}
	if m.DisplayCount < 0 {
		return ErrInvalidPayload
	}
	scores := []float64{
		m.DisplayAvg,
		m.Breakdown.Cleanliness,
		m.Breakdown.Comfort,
		m.Breakdown.Location,
		m.Breakdown.Host,
		m.Breakdown.Value,
	}
	for _, s := range scores {
		if s < 0 || s > 5 {
			return ErrInvalidPayload
		}
	}
	return nil
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
