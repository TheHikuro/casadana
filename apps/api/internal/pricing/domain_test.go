package pricing

import (
	"strings"
	"testing"
	"time"
)

func TestSettings_Validate(t *testing.T) {
	valid := Settings{VillaSlug: "casadana", BasePriceCents: 18500, MinNights: 3, CleaningFeeCents: 8000, ConciergeFeeCents: 5000}

	cases := []struct {
		name    string
		mutate  func(s *Settings)
		wantErr error
	}{
		{"valid", func(*Settings) {}, nil},
		{"zero fees and one night", func(s *Settings) {
			s.BasePriceCents, s.CleaningFeeCents, s.ConciergeFeeCents, s.MinNights = 0, 0, 0, 1
		}, nil},
		{"negative base price", func(s *Settings) { s.BasePriceCents = -1 }, ErrInvalidPayload},
		{"negative cleaning fee", func(s *Settings) { s.CleaningFeeCents = -1 }, ErrInvalidPayload},
		{"negative concierge fee", func(s *Settings) { s.ConciergeFeeCents = -1 }, ErrInvalidPayload},
		{"zero min nights", func(s *Settings) { s.MinNights = 0 }, ErrInvalidPayload},
		{"negative min nights", func(s *Settings) { s.MinNights = -2 }, ErrInvalidPayload},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := valid
			c.mutate(&s)
			if err := s.Validate(); err != c.wantErr {
				t.Fatalf("err = %v, want %v", err, c.wantErr)
			}
		})
	}
}

func TestSeasonRule_Validate(t *testing.T) {
	valid := SeasonRule{
		ID: "rule-1", VillaSlug: "casadana", Label: "Summer peak",
		Start: d("2026-07-01"), End: d("2026-08-31"), PriceCents: 25000,
	}

	cases := []struct {
		name    string
		mutate  func(r *SeasonRule)
		wantErr error
	}{
		{"valid", func(*SeasonRule) {}, nil},
		{"single day range", func(r *SeasonRule) { r.End = r.Start }, nil},
		{"free stay", func(r *SeasonRule) { r.PriceCents = 0 }, nil},
		{"empty label", func(r *SeasonRule) { r.Label = "" }, ErrInvalidPayload},
		{"blank label", func(r *SeasonRule) { r.Label = "   " }, ErrInvalidPayload},
		{"label at limit", func(r *SeasonRule) { r.Label = strings.Repeat("a", 120) }, nil},
		{"label too long", func(r *SeasonRule) { r.Label = strings.Repeat("a", 121) }, ErrInvalidPayload},
		{"end before start", func(r *SeasonRule) { r.End = d("2026-06-30") }, ErrInvalidPayload},
		{"negative price", func(r *SeasonRule) { r.PriceCents = -1 }, ErrInvalidPayload},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := valid
			c.mutate(&r)
			if err := r.Validate(); err != c.wantErr {
				t.Fatalf("err = %v, want %v", err, c.wantErr)
			}
		})
	}
}

func TestNewSeasonRule(t *testing.T) {
	rule, err := NewSeasonRule(NewSeasonRuleInput{
		VillaSlug:  "casadana",
		Label:      "  Summer peak  ",
		Start:      d("2026-07-01"),
		End:        d("2026-08-31"),
		PriceCents: 25000,
	})
	if err != nil {
		t.Fatalf("NewSeasonRule: %v", err)
	}
	if rule.ID == "" {
		t.Error("ID = empty, want a generated uuid")
	}
	if rule.Label != "Summer peak" {
		t.Errorf("Label = %q, want trimmed", rule.Label)
	}

	if _, err := NewSeasonRule(NewSeasonRuleInput{VillaSlug: "", Label: "x", Start: d("2026-07-01"), End: d("2026-07-02")}); err != ErrInvalidPayload {
		t.Fatalf("empty slug err = %v, want ErrInvalidPayload", err)
	}
	if _, err := NewSeasonRule(NewSeasonRuleInput{VillaSlug: "casadana", Label: " ", Start: d("2026-07-01"), End: d("2026-07-02")}); err != ErrInvalidPayload {
		t.Fatalf("blank label err = %v, want ErrInvalidPayload", err)
	}
}

func TestSeasonRule_Covers(t *testing.T) {
	rule := SeasonRule{Start: d("2026-07-01"), End: d("2026-07-31")}

	cases := []struct {
		date string
		want bool
	}{
		{"2026-06-30", false},
		{"2026-07-01", true}, // start is inclusive
		{"2026-07-15", true},
		{"2026-07-31", true}, // end is inclusive
		{"2026-08-01", false},
	}
	for _, c := range cases {
		t.Run(c.date, func(t *testing.T) {
			if got := rule.Covers(d(c.date)); got != c.want {
				t.Errorf("Covers(%s) = %v, want %v", c.date, got, c.want)
			}
		})
	}
}

func TestResolveNightly(t *testing.T) {
	settings := Settings{VillaSlug: "casadana", BasePriceCents: 18500, MinNights: 3}
	summer := SeasonRule{ID: "r1", VillaSlug: "casadana", Label: "Summer peak", Start: d("2026-07-01"), End: d("2026-08-31"), PriceCents: 25000}
	august := SeasonRule{ID: "r2", VillaSlug: "casadana", Label: "August premium", Start: d("2026-08-01"), End: d("2026-08-15"), PriceCents: 32000}
	cheap := SeasonRule{ID: "r3", VillaSlug: "casadana", Label: "Last minute", Start: d("2026-08-01"), End: d("2026-08-15"), PriceCents: 12000}

	cases := []struct {
		name      string
		date      string
		overrides []PriceOverride
		rules     []SeasonRule
		wantPrice int
		wantLabel string
	}{
		{
			name:      "falls back to base price",
			date:      "2026-05-10",
			wantPrice: 18500,
			wantLabel: StandardLabel,
		},
		{
			name:      "base price with rules that do not match",
			date:      "2026-05-10",
			rules:     []SeasonRule{summer, august},
			wantPrice: 18500,
			wantLabel: StandardLabel,
		},
		{
			name:      "single matching season rule",
			date:      "2026-07-04",
			rules:     []SeasonRule{summer, august},
			wantPrice: 25000,
			wantLabel: "Summer peak",
		},
		{
			name:      "highest matching season rule wins",
			date:      "2026-08-10",
			rules:     []SeasonRule{summer, august, cheap},
			wantPrice: 32000,
			wantLabel: "August premium",
		},
		{
			name:      "highest wins regardless of slice order",
			date:      "2026-08-10",
			rules:     []SeasonRule{august, cheap, summer},
			wantPrice: 32000,
			wantLabel: "August premium",
		},
		{
			name:      "override beats every season rule",
			date:      "2026-08-10",
			overrides: []PriceOverride{{VillaSlug: "casadana", Date: d("2026-08-10"), PriceCents: 9900}},
			rules:     []SeasonRule{summer, august},
			wantPrice: 9900,
			wantLabel: OverrideLabel,
		},
		{
			name:      "override beats the base price",
			date:      "2026-05-10",
			overrides: []PriceOverride{{VillaSlug: "casadana", Date: d("2026-05-10"), PriceCents: 9900}},
			wantPrice: 9900,
			wantLabel: OverrideLabel,
		},
		{
			name:      "override on another date is ignored",
			date:      "2026-08-10",
			overrides: []PriceOverride{{VillaSlug: "casadana", Date: d("2026-08-11"), PriceCents: 9900}},
			rules:     []SeasonRule{summer},
			wantPrice: 25000,
			wantLabel: "Summer peak",
		},
		{
			name:      "zero override still wins over the base price",
			date:      "2026-05-10",
			overrides: []PriceOverride{{VillaSlug: "casadana", Date: d("2026-05-10"), PriceCents: 0}},
			wantPrice: 0,
			wantLabel: OverrideLabel,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ResolveNightly(d(c.date), c.overrides, c.rules, settings)
			if got.PriceCents != c.wantPrice {
				t.Errorf("PriceCents = %d, want %d", got.PriceCents, c.wantPrice)
			}
			if got.Label != c.wantLabel {
				t.Errorf("Label = %q, want %q", got.Label, c.wantLabel)
			}
			if !got.Date.Equal(d(c.date)) {
				t.Errorf("Date = %s, want %s", got.Date, d(c.date))
			}
		})
	}
}

func TestResolveNightly_IgnoresClockPart(t *testing.T) {
	// Dates arrive from pgtype.Date, query strings and JSON bodies; only the
	// civil date may drive the match.
	noon := d("2026-08-10").Add(12 * time.Hour)
	got := ResolveNightly(noon,
		[]PriceOverride{{VillaSlug: "casadana", Date: d("2026-08-10").Add(23 * time.Hour), PriceCents: 9900}},
		nil, Settings{BasePriceCents: 18500})
	if got.PriceCents != 9900 || got.Label != OverrideLabel {
		t.Fatalf("got %+v, want the override to match", got)
	}
}
