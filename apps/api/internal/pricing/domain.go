package pricing

import (
	"errors"
	"time"
)

type PriceOverride struct {
	VillaSlug  string
	Date       time.Time
	PriceCents int
}

var (
	ErrUnknownVilla   = errors.New("unknown villa")
	ErrInvalidRange   = errors.New("from must be before to")
	ErrInvalidPayload = errors.New("invalid payload")
)
