package pricing

import (
	"context"
	"time"
)

type Service struct {
	repo  Repository
	allow VillaAllowlist
}

func NewService(repo Repository, allow VillaAllowlist) *Service {
	return &Service{repo: repo, allow: allow}
}

func (s *Service) ListOverrides(ctx context.Context, villaSlug string, from, to time.Time) ([]PriceOverride, error) {
	if !s.allow.IsKnown(villaSlug) {
		return nil, ErrUnknownVilla
	}
	if !to.After(from) {
		return nil, ErrInvalidRange
	}
	return s.repo.ListOverrides(ctx, villaSlug, from, to)
}

// UpsertOverrides bulk-sets price_cents for each date in `dates` for a villa.
// Returns the number of upserted rows on success.
func (s *Service) UpsertOverrides(ctx context.Context, villaSlug string, priceCents int, dates []time.Time) (int, error) {
	if !s.allow.IsKnown(villaSlug) {
		return 0, ErrUnknownVilla
	}
	if priceCents < 0 || len(dates) == 0 {
		return 0, ErrInvalidPayload
	}
	if err := s.repo.UpsertMany(ctx, villaSlug, priceCents, dates); err != nil {
		return 0, err
	}
	return len(dates), nil
}
