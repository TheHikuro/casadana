package audit

import (
	"context"
	"fmt"

	"github.com/TheHikuro/casadana/internal/villaslug"
)

type Service struct {
	repo  Repository
	clock Clock
}

func NewService(repo Repository, clock Clock) *Service {
	return &Service{repo: repo, clock: clock}
}

type RecordCommand struct {
	VillaSlug  string
	Type       EventType
	Message    string
	ActorEmail string
}

// Record appends one event to the log. Callers treat it as best-effort: the
// error is returned for logging, never to fail the change being recorded.
func (s *Service) Record(ctx context.Context, cmd RecordCommand) (*Event, error) {
	e, err := NewEvent(NewEventInput{
		VillaSlug:  cmd.VillaSlug,
		Type:       cmd.Type,
		Message:    cmd.Message,
		ActorEmail: cmd.ActorEmail,
		Now:        s.clock.Now(),
	})
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, e); err != nil {
		return nil, fmt.Errorf("audit: save event: %w", err)
	}
	return e, nil
}

type ListQuery struct {
	VillaSlug string
	Page      int
	Limit     int
}

// List returns a page of events ordered by created_at DESC. Page is 1-based,
// limit is clamped to [1, 100] and defaults to 20.
func (s *Service) List(ctx context.Context, q ListQuery) ([]Event, int, error) {
	if !villaslug.IsKnown(q.VillaSlug) {
		return nil, 0, ErrInvalidPayload
	}
	_, limit, offset := normalizePaging(q.Page, q.Limit)

	events, err := s.repo.List(ctx, q.VillaSlug, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("audit: list events: %w", err)
	}
	total, err := s.repo.Count(ctx, q.VillaSlug)
	if err != nil {
		return nil, 0, fmt.Errorf("audit: count events: %w", err)
	}
	return events, total, nil
}
