// Package audit records every change made from the admin dashboard as an
// append-only activity log, and serves it back most-recent-first to the
// dashboard's History screen.
package audit

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/TheHikuro/casadana/internal/villaslug"
)

type EventType string

const (
	TypeReservation EventType = "reservation"
	TypePricing     EventType = "pricing"
	TypeReview      EventType = "review"
	TypeOwner       EventType = "owner"
	TypeSystem      EventType = "system"
)

const maxMessageLen = 500

// Pagination defaults shared by the service and the handler that echoes the
// applied page/limit back to the client.
const (
	defaultLimit = 20
	maxLimit     = 100
)

type Event struct {
	ID         string
	VillaSlug  string
	Type       EventType
	Message    string
	ActorEmail string
	CreatedAt  time.Time
}

type NewEventInput struct {
	VillaSlug  string
	Type       EventType
	Message    string
	ActorEmail string
	Now        time.Time
}

var ErrInvalidPayload = errors.New("invalid audit event payload")

func (t EventType) IsValid() bool {
	switch t {
	case TypeReservation, TypePricing, TypeReview, TypeOwner, TypeSystem:
		return true
	default:
		return false
	}
}

// ParseEventType turns raw input (query string, JSON) into an EventType,
// rejecting anything outside the enum.
func ParseEventType(s string) (EventType, error) {
	t := EventType(strings.TrimSpace(s))
	if !t.IsValid() {
		return "", ErrInvalidPayload
	}
	return t, nil
}

func NewEvent(in NewEventInput) (*Event, error) {
	in.Message = strings.TrimSpace(in.Message)
	in.ActorEmail = strings.TrimSpace(in.ActorEmail)

	if !villaslug.IsKnown(in.VillaSlug) {
		return nil, ErrInvalidPayload
	}
	if !in.Type.IsValid() {
		return nil, ErrInvalidPayload
	}
	if in.Message == "" || len(in.Message) > maxMessageLen {
		return nil, ErrInvalidPayload
	}

	return &Event{
		ID:         uuid.NewString(),
		VillaSlug:  in.VillaSlug,
		Type:       in.Type,
		Message:    in.Message,
		ActorEmail: in.ActorEmail,
		CreatedAt:  in.Now,
	}, nil
}

// normalizePaging applies the 1-based page floor and the [1, 100] limit
// window, then derives the SQL offset. Shared by the service and the handler
// so the echoed page/limit always match the ones actually queried.
func normalizePaging(page, limit int) (normPage, normLimit, offset int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return page, limit, (page - 1) * limit
}
