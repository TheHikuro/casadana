package pricing

import (
	"context"
	"time"
)

type fakeRepo struct {
	overrides []PriceOverride
	listErr   error

	// settings holds only the villas that were actually configured; a missing
	// key mirrors the adapter turning pgx.ErrNoRows into the zero value.
	settings map[string]Settings

	rules []SeasonRule
}

func (f *fakeRepo) ListOverrides(_ context.Context, _ string, _, _ time.Time) ([]PriceOverride, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.overrides, nil
}

type fakeAllowlist struct {
	allowed map[string]bool
}

func (f fakeAllowlist) IsKnown(slug string) bool { return f.allowed[slug] }

func d(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func (f *fakeRepo) UpsertMany(_ context.Context, villaSlug string, priceCents int, dates []time.Time) error {
	for _, d := range dates {
		f.overrides = append(f.overrides, PriceOverride{
			VillaSlug:  villaSlug,
			Date:       d,
			PriceCents: priceCents,
		})
	}
	return nil
}

func (f *fakeRepo) GetSettings(_ context.Context, villaSlug string) (Settings, error) {
	if s, ok := f.settings[villaSlug]; ok {
		return s, nil
	}
	return Settings{VillaSlug: villaSlug}, nil
}

func (f *fakeRepo) SaveSettings(_ context.Context, s Settings) (Settings, error) {
	if f.settings == nil {
		f.settings = map[string]Settings{}
	}
	f.settings[s.VillaSlug] = s
	return s, nil
}

func (f *fakeRepo) ListSeasonRules(_ context.Context, villaSlug string) ([]SeasonRule, error) {
	out := make([]SeasonRule, 0, len(f.rules))
	for _, r := range f.rules {
		if r.VillaSlug == villaSlug {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeRepo) GetSeasonRule(_ context.Context, id string) (SeasonRule, error) {
	for _, r := range f.rules {
		if r.ID == id {
			return r, nil
		}
	}
	return SeasonRule{}, ErrRuleNotFound
}

func (f *fakeRepo) InsertSeasonRule(_ context.Context, r SeasonRule) (SeasonRule, error) {
	f.rules = append(f.rules, r)
	return r, nil
}

func (f *fakeRepo) UpdateSeasonRule(_ context.Context, r SeasonRule) (SeasonRule, error) {
	for i := range f.rules {
		if f.rules[i].ID == r.ID {
			f.rules[i] = r
			return r, nil
		}
	}
	return SeasonRule{}, ErrRuleNotFound
}

func (f *fakeRepo) DeleteSeasonRule(_ context.Context, id string) error {
	for i := range f.rules {
		if f.rules[i].ID == id {
			f.rules = append(f.rules[:i], f.rules[i+1:]...)
			return nil
		}
	}
	return ErrRuleNotFound
}

type recordedEvent struct {
	villaSlug string
	message   string
}

type fakeRecorder struct {
	events []recordedEvent
	err    error
}

func (f *fakeRecorder) Record(_ context.Context, villaSlug, message string) error {
	if f.err != nil {
		return f.err
	}
	f.events = append(f.events, recordedEvent{villaSlug: villaSlug, message: message})
	return nil
}
