package pricing

import (
	"context"
	"testing"
	"time"
)

func newSvc(repo Repository, allow VillaAllowlist) *Service {
	return NewService(repo, allow)
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
