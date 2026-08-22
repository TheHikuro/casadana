package pricing

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type PriceOverride struct {
	VillaSlug  string
	Date       time.Time
	PriceCents int
}

// Settings is the per-villa base rate and the fees added on top of a stay.
// A villa that was never configured reads back as the zero value.
type Settings struct {
	VillaSlug         string
	BasePriceCents    int
	MinNights         int
	CleaningFeeCents  int
	ConciergeFeeCents int
}

// SeasonRule prices a named date range ("Summer peak"). Both bounds are
// inclusive, mirroring the season_rules CHECK (end_date >= start_date).
type SeasonRule struct {
	ID         string
	VillaSlug  string
	Label      string
	Start      time.Time
	End        time.Time
	PriceCents int
}

type NewSeasonRuleInput struct {
	VillaSlug  string
	Label      string
	Start      time.Time
	End        time.Time
	PriceCents int
}

var (
	ErrUnknownVilla   = errors.New("unknown villa")
	ErrInvalidRange   = errors.New("from must be before to")
	ErrRangeTooLarge  = errors.New("date window too large")
	ErrInvalidPayload = errors.New("invalid payload")
	ErrRuleNotFound   = errors.New("season rule not found")
)

// StandardLabel is the label reported when neither an override nor a season
// rule matches and the nightly price falls back to the base rate.
const StandardLabel = "Standard"

// OverrideLabel is the label reported for a per-date price_overrides row,
// which has no name of its own.
const OverrideLabel = "Custom"

func (s Settings) Validate() error {
	if s.BasePriceCents < 0 || s.CleaningFeeCents < 0 || s.ConciergeFeeCents < 0 {
		return ErrInvalidPayload
	}
	if s.MinNights < 1 {
		return ErrInvalidPayload
	}
	return nil
}

func (r SeasonRule) Validate() error {
	if strings.TrimSpace(r.Label) == "" || len(r.Label) > 120 {
		return ErrInvalidPayload
	}
	if r.End.Before(r.Start) {
		return ErrInvalidPayload
	}
	if r.PriceCents < 0 {
		return ErrInvalidPayload
	}
	return nil
}

func NewSeasonRule(in NewSeasonRuleInput) (*SeasonRule, error) {
	rule := SeasonRule{
		ID:         uuid.NewString(),
		VillaSlug:  in.VillaSlug,
		Label:      strings.TrimSpace(in.Label),
		Start:      Day(in.Start),
		End:        Day(in.End),
		PriceCents: in.PriceCents,
	}
	if rule.VillaSlug == "" {
		return nil, ErrInvalidPayload
	}
	if err := rule.Validate(); err != nil {
		return nil, err
	}
	return &rule, nil
}

// Covers reports whether the rule applies to date. Both bounds are inclusive.
func (r SeasonRule) Covers(date time.Time) bool {
	d := Day(date)
	return !d.Before(Day(r.Start)) && !d.After(Day(r.End))
}

// NightlyPrice is the effective price for a single night plus the label
// explaining where that price came from.
type NightlyPrice struct {
	Date       time.Time
	PriceCents int
	Label      string
}

// ResolveNightly resolves the effective nightly price for date, in the order
// the admin dashboard promises: a per-date override wins over any season rule,
// and when several season rules overlap the most expensive one wins (the
// "highest match wins" rule from the design). With no match at all the villa's
// base rate applies under the "Standard" label.
func ResolveNightly(date time.Time, overrides []PriceOverride, rules []SeasonRule, settings Settings) NightlyPrice {
	day := Day(date)

	for _, o := range overrides {
		if Day(o.Date).Equal(day) {
			return NightlyPrice{Date: day, PriceCents: o.PriceCents, Label: OverrideLabel}
		}
	}

	best := -1
	for i, rule := range rules {
		if !rule.Covers(day) {
			continue
		}
		if best == -1 || rule.PriceCents > rules[best].PriceCents {
			best = i
		}
	}
	if best >= 0 {
		return NightlyPrice{Date: day, PriceCents: rules[best].PriceCents, Label: rules[best].Label}
	}

	return NightlyPrice{Date: day, PriceCents: settings.BasePriceCents, Label: StandardLabel}
}

// Day drops the clock part so dates coming from pgtype.Date, query strings and
// JSON bodies all compare on the same footing.
func Day(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
