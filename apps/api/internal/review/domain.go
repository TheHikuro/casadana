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

type Review struct {
	ID         string
	BookingID  string
	VillaSlug  string
	AuthorName string
	Rating     int
	Body       string
	Status     Status
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
	if in.AuthorName == "" || len(in.AuthorName) > 120 {
		return nil, ErrInvalidPayload
	}
	if in.Rating < 1 || in.Rating > 5 {
		return nil, ErrInvalidPayload
	}
	if len(in.Body) > 2000 {
		return nil, ErrInvalidPayload
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
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}
