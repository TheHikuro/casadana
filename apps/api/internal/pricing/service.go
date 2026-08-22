package pricing

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

type Service struct {
	repo   Repository
	allow  VillaAllowlist
	events EventRecorder
}

// NewService wires the pricing use cases. events may be nil, in which case
// mutations simply go unrecorded.
func NewService(repo Repository, allow VillaAllowlist, events EventRecorder) *Service {
	return &Service{repo: repo, allow: allow, events: events}
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
	s.record(ctx, villaSlug, fmt.Sprintf("Nightly price set to %s on %d date(s)", euros(priceCents), len(dates)))
	return len(dates), nil
}

// Calendar is what the public booking panel needs to price a stay: the
// effective price of every night in the window, plus the raw per-date
// overrides that window contains.
type Calendar struct {
	Overrides []PriceOverride
	Nights    []NightlyPrice
}

// maxCalendarDays bounds how many nights one request can expand into. The
// endpoint is public and unauthenticated, and the booking panel never asks for
// more than a few months, so a wider window is a mistake or an abuse.
const maxCalendarDays = 400

// ResolveCalendar prices every day in [from, to). Season rules and the base
// rate are resolved here rather than in the client so the public site, the
// back-office and any future consumer all agree on what a night costs.
func (s *Service) ResolveCalendar(ctx context.Context, villaSlug string, from, to time.Time) (Calendar, error) {
	// Reuses ListOverrides for the allowlist and range checks.
	overrides, err := s.ListOverrides(ctx, villaSlug, from, to)
	if err != nil {
		return Calendar{}, err
	}

	start, end := Day(from), Day(to)
	days := int(end.Sub(start).Hours() / 24)
	if days > maxCalendarDays {
		return Calendar{}, ErrRangeTooLarge
	}

	rules, err := s.repo.ListSeasonRules(ctx, villaSlug)
	if err != nil {
		return Calendar{}, err
	}
	settings, err := s.repo.GetSettings(ctx, villaSlug)
	if err != nil {
		return Calendar{}, err
	}

	nights := make([]NightlyPrice, 0, days)
	for day := start; day.Before(end); day = day.AddDate(0, 0, 1) {
		nights = append(nights, ResolveNightly(day, overrides, rules, settings))
	}
	return Calendar{Overrides: overrides, Nights: nights}, nil
}

func (s *Service) GetSettings(ctx context.Context, villaSlug string) (Settings, error) {
	if !s.allow.IsKnown(villaSlug) {
		return Settings{}, ErrUnknownVilla
	}
	settings, err := s.repo.GetSettings(ctx, villaSlug)
	if err != nil {
		return Settings{}, err
	}
	// A villa that exists but was never configured reads back as zeros, not
	// as a 404: the admin dashboard opens on an empty-but-valid form.
	settings.VillaSlug = villaSlug
	return settings, nil
}

func (s *Service) SaveSettings(ctx context.Context, in Settings) (Settings, error) {
	if !s.allow.IsKnown(in.VillaSlug) {
		return Settings{}, ErrUnknownVilla
	}
	if err := in.Validate(); err != nil {
		return Settings{}, err
	}
	saved, err := s.repo.SaveSettings(ctx, in)
	if err != nil {
		return Settings{}, err
	}
	s.record(ctx, saved.VillaSlug, fmt.Sprintf(
		"Base price set to %s, minimum stay %d night(s), cleaning fee %s, concierge fee %s",
		euros(saved.BasePriceCents), saved.MinNights, euros(saved.CleaningFeeCents), euros(saved.ConciergeFeeCents),
	))
	return saved, nil
}

func (s *Service) ListSeasonRules(ctx context.Context, villaSlug string) ([]SeasonRule, error) {
	if !s.allow.IsKnown(villaSlug) {
		return nil, ErrUnknownVilla
	}
	return s.repo.ListSeasonRules(ctx, villaSlug)
}

type CreateSeasonRuleCommand struct {
	VillaSlug  string
	Label      string
	Start      time.Time
	End        time.Time
	PriceCents int
}

func (s *Service) CreateSeasonRule(ctx context.Context, cmd CreateSeasonRuleCommand) (*SeasonRule, error) {
	if !s.allow.IsKnown(cmd.VillaSlug) {
		return nil, ErrUnknownVilla
	}
	rule, err := NewSeasonRule(NewSeasonRuleInput{
		VillaSlug:  cmd.VillaSlug,
		Label:      cmd.Label,
		Start:      cmd.Start,
		End:        cmd.End,
		PriceCents: cmd.PriceCents,
	})
	if err != nil {
		return nil, err
	}
	saved, err := s.repo.InsertSeasonRule(ctx, *rule)
	if err != nil {
		return nil, err
	}
	s.record(ctx, saved.VillaSlug, fmt.Sprintf("Season rule %q added at %s (%s → %s)",
		saved.Label, euros(saved.PriceCents), ymd(saved.Start), ymd(saved.End)))
	return &saved, nil
}

// PatchSeasonRuleCommand carries only the fields the caller sent; a nil field
// leaves the stored value untouched.
type PatchSeasonRuleCommand struct {
	Label      *string
	Start      *time.Time
	End        *time.Time
	PriceCents *int
}

func (s *Service) UpdateSeasonRule(ctx context.Context, id string, cmd PatchSeasonRuleCommand) (*SeasonRule, error) {
	// Read first so the merged rule can be validated as a whole: a patch that
	// only moves start_date can still invert the range.
	current, err := s.repo.GetSeasonRule(ctx, id)
	if err != nil {
		return nil, err
	}

	updated := current
	if cmd.Label != nil {
		updated.Label = strings.TrimSpace(*cmd.Label)
	}
	if cmd.Start != nil {
		updated.Start = Day(*cmd.Start)
	}
	if cmd.End != nil {
		updated.End = Day(*cmd.End)
	}
	if cmd.PriceCents != nil {
		updated.PriceCents = *cmd.PriceCents
	}
	if err := updated.Validate(); err != nil {
		return nil, err
	}

	saved, err := s.repo.UpdateSeasonRule(ctx, updated)
	if err != nil {
		return nil, err
	}
	s.record(ctx, saved.VillaSlug, fmt.Sprintf("Season rule %q updated to %s (%s → %s)",
		saved.Label, euros(saved.PriceCents), ymd(saved.Start), ymd(saved.End)))
	return &saved, nil
}

func (s *Service) DeleteSeasonRule(ctx context.Context, id string) error {
	// Fetched for the audit line — the caller only knows the rule's id.
	rule, err := s.repo.GetSeasonRule(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteSeasonRule(ctx, id); err != nil {
		return err
	}
	s.record(ctx, rule.VillaSlug, fmt.Sprintf("Season rule %q removed", rule.Label))
	return nil
}

// record appends an audit line best-effort: no recorder configured is fine,
// and a failing recorder must never turn a successful mutation into a 500.
func (s *Service) record(ctx context.Context, villaSlug, message string) {
	if s.events == nil {
		return
	}
	if err := s.events.Record(ctx, villaSlug, message); err != nil {
		slog.WarnContext(ctx, "pricing audit event failed", "villa_slug", villaSlug, "err", err.Error())
	}
}

// euros renders cents the way a human reads them in the audit trail:
// 18500 → "€185", 18550 → "€185.50".
func euros(cents int) string {
	if cents%100 == 0 {
		return fmt.Sprintf("€%d", cents/100)
	}
	return fmt.Sprintf("€%d.%02d", cents/100, cents%100)
}

func ymd(t time.Time) string { return t.Format("2006-01-02") }
