package pricing

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// newSvc builds a service without an audit recorder — the nil recorder must be
// tolerated, which keeps the tests that aren't about auditing simple.
func newSvc(repo Repository, allow VillaAllowlist) *Service {
	return NewService(repo, allow, nil)
}

func newSvcWithRecorder(repo Repository, allow VillaAllowlist, events EventRecorder) *Service {
	return NewService(repo, allow, events)
}

func casadanaOnly() fakeAllowlist {
	return fakeAllowlist{allowed: map[string]bool{"casadana": true}}
}

func TestListOverrides_Happy(t *testing.T) {
	repo := &fakeRepo{
		overrides: []PriceOverride{
			{VillaSlug: "casadana", Date: d("2026-07-04"), PriceCents: 25000},
			{VillaSlug: "casadana", Date: d("2026-07-05"), PriceCents: 25000},
		},
	}
	allow := fakeAllowlist{allowed: map[string]bool{"casadana": true}}
	svc := newSvc(repo, allow)

	out, err := svc.ListOverrides(context.Background(), "casadana", d("2026-07-01"), d("2026-08-01"))
	if err != nil {
		t.Fatalf("ListOverrides: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("len = %d, want 2", len(out))
	}
}

func TestListOverrides_UnknownVilla(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, fakeAllowlist{allowed: map[string]bool{}})

	_, err := svc.ListOverrides(context.Background(), "ghost", d("2026-07-01"), d("2026-08-01"))
	if err != ErrUnknownVilla {
		t.Fatalf("err = %v, want ErrUnknownVilla", err)
	}
}

func TestListOverrides_InvalidRange(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, fakeAllowlist{allowed: map[string]bool{"casadana": true}})

	cases := []struct {
		name     string
		from, to string
	}{
		{"to before from", "2026-08-01", "2026-07-01"},
		{"to equals from", "2026-07-01", "2026-07-01"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := svc.ListOverrides(context.Background(), "casadana", d(c.from), d(c.to))
			if err != ErrInvalidRange {
				t.Fatalf("err = %v, want ErrInvalidRange", err)
			}
		})
	}
}

// The whole point of ResolveCalendar: a season rule configured in the
// back-office has to reach the public quote, and a per-date override has to
// beat it. Everything else falls back to the base rate.
func TestResolveCalendar_AppliesOverridesRulesAndBase(t *testing.T) {
	repo := &fakeRepo{
		overrides: []PriceOverride{
			{VillaSlug: "casadana", Date: d("2026-07-03"), PriceCents: 40000},
		},
		rules: []SeasonRule{
			{ID: "r1", VillaSlug: "casadana", Label: "Summer", Start: d("2026-07-02"), End: d("2026-07-04"), PriceCents: 20000},
		},
		settings: map[string]Settings{
			"casadana": {VillaSlug: "casadana", BasePriceCents: 8500, MinNights: 3},
		},
	}
	svc := newSvc(repo, casadanaOnly())

	cal, err := svc.ResolveCalendar(context.Background(), "casadana", d("2026-07-01"), d("2026-07-05"))
	if err != nil {
		t.Fatalf("ResolveCalendar: %v", err)
	}
	if len(cal.Nights) != 4 {
		t.Fatalf("len(nights) = %d, want 4 (one per day in the window)", len(cal.Nights))
	}
	want := []NightlyPrice{
		{Date: d("2026-07-01"), PriceCents: 8500, Label: StandardLabel},
		{Date: d("2026-07-02"), PriceCents: 20000, Label: "Summer"},
		{Date: d("2026-07-03"), PriceCents: 40000, Label: OverrideLabel},
		{Date: d("2026-07-04"), PriceCents: 20000, Label: "Summer"},
	}
	for i, w := range want {
		got := cal.Nights[i]
		if !got.Date.Equal(w.Date) || got.PriceCents != w.PriceCents || got.Label != w.Label {
			t.Errorf("night %d = %v, want %v", i, got, w)
		}
	}
	if len(cal.Overrides) != 1 {
		t.Errorf("len(overrides) = %d, want 1", len(cal.Overrides))
	}
}

func TestResolveCalendar_RejectsBadWindow(t *testing.T) {
	svc := newSvc(&fakeRepo{}, casadanaOnly())

	cases := []struct {
		name     string
		slug     string
		from, to string
		want     error
	}{
		{"unknown villa", "ghost", "2026-07-01", "2026-08-01", ErrUnknownVilla},
		{"inverted range", "casadana", "2026-08-01", "2026-07-01", ErrInvalidRange},
		{"wider than the cap", "casadana", "2026-01-01", "2028-01-01", ErrRangeTooLarge},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := svc.ResolveCalendar(context.Background(), c.slug, d(c.from), d(c.to))
			if !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
		})
	}
}

// A window exactly at the cap is still served — the guard is on wider ones.
func TestResolveCalendar_AllowsWindowAtCap(t *testing.T) {
	svc := newSvc(&fakeRepo{}, casadanaOnly())

	from := d("2026-01-01")
	cal, err := svc.ResolveCalendar(context.Background(), "casadana", from, from.AddDate(0, 0, maxCalendarDays))
	if err != nil {
		t.Fatalf("ResolveCalendar: %v", err)
	}
	if len(cal.Nights) != maxCalendarDays {
		t.Errorf("len(nights) = %d, want %d", len(cal.Nights), maxCalendarDays)
	}
}

func TestUpsertOverrides_Happy(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, fakeAllowlist{allowed: map[string]bool{"casadana": true}})

	count, err := svc.UpsertOverrides(context.Background(), "casadana", 25000, []time.Time{
		d("2026-07-04"), d("2026-07-05"),
	})
	if err != nil {
		t.Fatalf("UpsertOverrides: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestUpsertOverrides_UnknownVilla(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{}})
	_, err := svc.UpsertOverrides(context.Background(), "ghost", 100, []time.Time{d("2026-07-04")})
	if err != ErrUnknownVilla {
		t.Fatalf("err = %v, want ErrUnknownVilla", err)
	}
}

func TestUpsertOverrides_NegativePrice(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	_, err := svc.UpsertOverrides(context.Background(), "casadana", -1, []time.Time{d("2026-07-04")})
	if err != ErrInvalidPayload {
		t.Fatalf("err = %v, want ErrInvalidPayload", err)
	}
}

func TestUpsertOverrides_EmptyDates(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	_, err := svc.UpsertOverrides(context.Background(), "casadana", 100, nil)
	if err != ErrInvalidPayload {
		t.Fatalf("err = %v, want ErrInvalidPayload", err)
	}
}

func TestGetSettings_NeverConfigured(t *testing.T) {
	// A known villa with no villa_pricing_settings row is a zero-valued
	// Settings, not a 404.
	svc := newSvc(&fakeRepo{}, casadanaOnly())

	got, err := svc.GetSettings(context.Background(), "casadana")
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	want := Settings{VillaSlug: "casadana"}
	if got != want {
		t.Errorf("settings = %+v, want %+v", got, want)
	}
}

func TestGetSettings_Configured(t *testing.T) {
	stored := Settings{VillaSlug: "casadana", BasePriceCents: 18500, MinNights: 3, CleaningFeeCents: 8000, ConciergeFeeCents: 5000}
	svc := newSvc(&fakeRepo{settings: map[string]Settings{"casadana": stored}}, casadanaOnly())

	got, err := svc.GetSettings(context.Background(), "casadana")
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got != stored {
		t.Errorf("settings = %+v, want %+v", got, stored)
	}
}

func TestGetSettings_UnknownVilla(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{}})
	if _, err := svc.GetSettings(context.Background(), "ghost"); err != ErrUnknownVilla {
		t.Fatalf("err = %v, want ErrUnknownVilla", err)
	}
}

func TestSaveSettings(t *testing.T) {
	valid := Settings{VillaSlug: "casadana", BasePriceCents: 18500, MinNights: 3, CleaningFeeCents: 8000, ConciergeFeeCents: 5000}

	cases := []struct {
		name    string
		in      Settings
		wantErr error
	}{
		{"happy", valid, nil},
		{"unknown villa", Settings{VillaSlug: "ghost", MinNights: 1}, ErrUnknownVilla},
		{"negative base price", Settings{VillaSlug: "casadana", BasePriceCents: -1, MinNights: 1}, ErrInvalidPayload},
		{"zero min nights", Settings{VillaSlug: "casadana", MinNights: 0}, ErrInvalidPayload},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := newSvc(&fakeRepo{}, casadanaOnly())
			got, err := svc.SaveSettings(context.Background(), c.in)
			if err != c.wantErr {
				t.Fatalf("err = %v, want %v", err, c.wantErr)
			}
			if c.wantErr == nil && got != c.in {
				t.Errorf("saved = %+v, want %+v", got, c.in)
			}
		})
	}
}

func TestSaveSettings_RecordsEvent(t *testing.T) {
	events := &fakeRecorder{}
	svc := newSvcWithRecorder(&fakeRepo{}, casadanaOnly(), events)

	_, err := svc.SaveSettings(context.Background(), Settings{VillaSlug: "casadana", BasePriceCents: 18500, MinNights: 3})
	if err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	if len(events.events) != 1 {
		t.Fatalf("events = %d, want 1", len(events.events))
	}
	if events.events[0].villaSlug != "casadana" {
		t.Errorf("villa_slug = %q, want casadana", events.events[0].villaSlug)
	}
	if !strings.Contains(events.events[0].message, "€185") {
		t.Errorf("message = %q, want it to mention €185", events.events[0].message)
	}
}

func TestSaveSettings_RecorderFailureIsNotFatal(t *testing.T) {
	svc := newSvcWithRecorder(&fakeRepo{}, casadanaOnly(), &fakeRecorder{err: errors.New("audit down")})

	if _, err := svc.SaveSettings(context.Background(), Settings{VillaSlug: "casadana", MinNights: 1}); err != nil {
		t.Fatalf("SaveSettings: %v, want the mutation to succeed anyway", err)
	}
}

func TestCreateSeasonRule(t *testing.T) {
	cases := []struct {
		name    string
		cmd     CreateSeasonRuleCommand
		wantErr error
	}{
		{
			name: "happy",
			cmd: CreateSeasonRuleCommand{VillaSlug: "casadana", Label: "Summer peak",
				Start: d("2026-07-01"), End: d("2026-08-31"), PriceCents: 25000},
		},
		{
			name:    "unknown villa",
			cmd:     CreateSeasonRuleCommand{VillaSlug: "ghost", Label: "x", Start: d("2026-07-01"), End: d("2026-07-02")},
			wantErr: ErrUnknownVilla,
		},
		{
			name:    "empty label",
			cmd:     CreateSeasonRuleCommand{VillaSlug: "casadana", Label: "  ", Start: d("2026-07-01"), End: d("2026-07-02")},
			wantErr: ErrInvalidPayload,
		},
		{
			name:    "end before start",
			cmd:     CreateSeasonRuleCommand{VillaSlug: "casadana", Label: "x", Start: d("2026-07-02"), End: d("2026-07-01")},
			wantErr: ErrInvalidPayload,
		},
		{
			name:    "negative price",
			cmd:     CreateSeasonRuleCommand{VillaSlug: "casadana", Label: "x", Start: d("2026-07-01"), End: d("2026-07-02"), PriceCents: -1},
			wantErr: ErrInvalidPayload,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := newSvc(&fakeRepo{}, casadanaOnly())
			rule, err := svc.CreateSeasonRule(context.Background(), c.cmd)
			if err != c.wantErr {
				t.Fatalf("err = %v, want %v", err, c.wantErr)
			}
			if c.wantErr == nil && (rule.ID == "" || rule.Label != "Summer peak") {
				t.Errorf("unexpected rule: %+v", rule)
			}
		})
	}
}

func TestUpdateSeasonRule_PartialPatch(t *testing.T) {
	stored := SeasonRule{ID: "r1", VillaSlug: "casadana", Label: "Summer peak",
		Start: d("2026-07-01"), End: d("2026-08-31"), PriceCents: 25000}
	repo := &fakeRepo{rules: []SeasonRule{stored}}
	svc := newSvc(repo, casadanaOnly())

	price := 30000
	got, err := svc.UpdateSeasonRule(context.Background(), "r1", PatchSeasonRuleCommand{PriceCents: &price})
	if err != nil {
		t.Fatalf("UpdateSeasonRule: %v", err)
	}
	if got.PriceCents != 30000 {
		t.Errorf("price_cents = %d, want 30000", got.PriceCents)
	}
	if got.Label != "Summer peak" || !got.Start.Equal(stored.Start) || !got.End.Equal(stored.End) {
		t.Errorf("untouched fields changed: %+v", got)
	}
}

func TestUpdateSeasonRule_MergedRangeIsValidated(t *testing.T) {
	// Moving only start_date can still invert the stored range.
	repo := &fakeRepo{rules: []SeasonRule{{ID: "r1", VillaSlug: "casadana", Label: "Summer peak",
		Start: d("2026-07-01"), End: d("2026-07-31"), PriceCents: 25000}}}
	svc := newSvc(repo, casadanaOnly())

	start := d("2026-09-01")
	if _, err := svc.UpdateSeasonRule(context.Background(), "r1", PatchSeasonRuleCommand{Start: &start}); err != ErrInvalidPayload {
		t.Fatalf("err = %v, want ErrInvalidPayload", err)
	}
}

func TestUpdateSeasonRule_NotFound(t *testing.T) {
	svc := newSvc(&fakeRepo{}, casadanaOnly())
	if _, err := svc.UpdateSeasonRule(context.Background(), "missing", PatchSeasonRuleCommand{}); err != ErrRuleNotFound {
		t.Fatalf("err = %v, want ErrRuleNotFound", err)
	}
}

func TestDeleteSeasonRule(t *testing.T) {
	repo := &fakeRepo{rules: []SeasonRule{{ID: "r1", VillaSlug: "casadana", Label: "Summer peak",
		Start: d("2026-07-01"), End: d("2026-07-31"), PriceCents: 25000}}}
	events := &fakeRecorder{}
	svc := newSvcWithRecorder(repo, casadanaOnly(), events)

	if err := svc.DeleteSeasonRule(context.Background(), "r1"); err != nil {
		t.Fatalf("DeleteSeasonRule: %v", err)
	}
	if len(repo.rules) != 0 {
		t.Errorf("rules = %d, want 0", len(repo.rules))
	}
	if len(events.events) != 1 || !strings.Contains(events.events[0].message, `"Summer peak" removed`) {
		t.Errorf("unexpected events: %+v", events.events)
	}
}

func TestDeleteSeasonRule_NotFound(t *testing.T) {
	svc := newSvc(&fakeRepo{}, casadanaOnly())
	if err := svc.DeleteSeasonRule(context.Background(), "missing"); err != ErrRuleNotFound {
		t.Fatalf("err = %v, want ErrRuleNotFound", err)
	}
}

func TestListSeasonRules_UnknownVilla(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{}})
	if _, err := svc.ListSeasonRules(context.Background(), "ghost"); err != ErrUnknownVilla {
		t.Fatalf("err = %v, want ErrUnknownVilla", err)
	}
}

func TestUpsertOverrides_RecordsEvent(t *testing.T) {
	events := &fakeRecorder{}
	svc := newSvcWithRecorder(&fakeRepo{}, casadanaOnly(), events)

	if _, err := svc.UpsertOverrides(context.Background(), "casadana", 25000, []time.Time{d("2026-07-04")}); err != nil {
		t.Fatalf("UpsertOverrides: %v", err)
	}
	if len(events.events) != 1 || !strings.Contains(events.events[0].message, "€250") {
		t.Errorf("unexpected events: %+v", events.events)
	}
}

func TestEuros(t *testing.T) {
	cases := []struct {
		cents int
		want  string
	}{
		{0, "€0"},
		{18500, "€185"},
		{18550, "€185.50"},
		{18505, "€185.05"},
	}
	for _, c := range cases {
		if got := euros(c.cents); got != c.want {
			t.Errorf("euros(%d) = %q, want %q", c.cents, got, c.want)
		}
	}
}
