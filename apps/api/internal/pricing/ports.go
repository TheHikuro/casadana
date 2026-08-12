package pricing

import (
	"context"
	"time"
)

type Repository interface {
	ListOverrides(ctx context.Context, villaSlug string, from, to time.Time) ([]PriceOverride, error)
	UpsertMany(ctx context.Context, villaSlug string, priceCents int, dates []time.Time) error

	// GetSettings returns the zero value (not an error) for a villa that has
	// never been configured.
	GetSettings(ctx context.Context, villaSlug string) (Settings, error)
	SaveSettings(ctx context.Context, s Settings) (Settings, error)

	ListSeasonRules(ctx context.Context, villaSlug string) ([]SeasonRule, error)
	GetSeasonRule(ctx context.Context, id string) (SeasonRule, error)
	InsertSeasonRule(ctx context.Context, r SeasonRule) (SeasonRule, error)
	UpdateSeasonRule(ctx context.Context, r SeasonRule) (SeasonRule, error)
	DeleteSeasonRule(ctx context.Context, id string) error
}

type VillaAllowlist interface {
	IsKnown(slug string) bool
}

// EventRecorder is the pricing slice's own view of the audit trail: it only
// needs to append a human-readable line for a villa. Declared here (rather
// than imported) so pricing stays independent of whichever module writes the
// events. A nil recorder is tolerated and recording failures never fail the
// mutation that triggered them.
type EventRecorder interface {
	Record(ctx context.Context, villaSlug, message string) error
}
