package review

import (
	"context"
	"math"
	"sort"
	"time"
)

type fakeRepo struct {
	saved   []Review
	saveErr error
}

func (f *fakeRepo) Save(_ context.Context, r *Review) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	for _, existing := range f.saved {
		// Only booking-backed reviews are unique per booking; admin-authored
		// ones carry no booking id at all.
		if r.BookingID != "" && existing.BookingID == r.BookingID {
			return ErrAlreadyReviewed
		}
	}
	f.saved = append(f.saved, *r)
	return nil
}

func (f *fakeRepo) ListByVillaAndStatus(_ context.Context, slug string, status *Status) ([]Review, error) {
	out := []Review{}
	for _, r := range f.saved {
		if r.VillaSlug != slug {
			continue
		}
		if status != nil && r.Status != *status {
			continue
		}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Featured && !out[j].Featured })
	return out, nil
}

func (f *fakeRepo) Get(_ context.Context, id string) (*Review, error) {
	for i := range f.saved {
		if f.saved[i].ID == id {
			r := f.saved[i]
			return &r, nil
		}
	}
	return nil, ErrNotFound
}

func (f *fakeRepo) Update(_ context.Context, id string, patch UpdatePatch) (*Review, error) {
	for i := range f.saved {
		if f.saved[i].ID != id {
			continue
		}
		if patch.Status != nil {
			f.saved[i].Status = *patch.Status
		}
		if patch.Featured != nil {
			f.saved[i].Featured = *patch.Featured
		}
		if patch.Meta != nil {
			f.saved[i].Meta = *patch.Meta
		}
		if patch.Source != nil {
			f.saved[i].Source = *patch.Source
		}
		if patch.Body != nil {
			f.saved[i].Body = *patch.Body
		}
		if patch.Rating != nil {
			f.saved[i].Rating = *patch.Rating
		}
		for _, c := range []struct {
			in  *float64
			out **float64
		}{
			{patch.Categories.Cleanliness, &f.saved[i].Categories.Cleanliness},
			{patch.Categories.Comfort, &f.saved[i].Categories.Comfort},
			{patch.Categories.Location, &f.saved[i].Categories.Location},
			{patch.Categories.Host, &f.saved[i].Categories.Host},
			{patch.Categories.Value, &f.saved[i].Categories.Value},
		} {
			if c.in != nil {
				*c.out = c.in
			}
		}
		r := f.saved[i]
		return &r, nil
	}
	return nil, ErrNotFound
}

func (f *fakeRepo) Delete(_ context.Context, id string) error {
	for i, r := range f.saved {
		if r.ID == id {
			f.saved = append(f.saved[:i], f.saved[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

// GetAggregate mirrors what the SQL does: average over the villa's approved
// reviews only, and let an unscored category fall out of its own average rather
// than counting as zero.
func (f *fakeRepo) GetAggregate(_ context.Context, slug string) (ReviewMeta, error) {
	meta := ReviewMeta{VillaSlug: slug}
	var ratingSum float64
	cleanliness, comfort, location, host, value := &avg{}, &avg{}, &avg{}, &avg{}, &avg{}
	for _, r := range f.saved {
		if r.VillaSlug != slug || r.Status != StatusApproved {
			continue
		}
		meta.DisplayCount++
		ratingSum += float64(r.Rating)
		cleanliness.add(r.Categories.Cleanliness)
		comfort.add(r.Categories.Comfort)
		location.add(r.Categories.Location)
		host.add(r.Categories.Host)
		value.add(r.Categories.Value)
	}
	if meta.DisplayCount > 0 {
		meta.DisplayAvg = round2(ratingSum / float64(meta.DisplayCount))
	}
	meta.Breakdown = Breakdown{
		Cleanliness: cleanliness.mean(),
		Comfort:     comfort.mean(),
		Location:    location.mean(),
		Host:        host.mean(),
		Value:       value.mean(),
	}
	return meta, nil
}

type avg struct {
	sum float64
	n   int
}

func (a *avg) add(v *float64) {
	if v == nil {
		return
	}
	a.sum += *v
	a.n++
}

func (a *avg) mean() *float64 {
	if a.n == 0 {
		return nil
	}
	m := round2(a.sum / float64(a.n))
	return &m
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }

type recordedEvent struct {
	villaSlug string
	message   string
}

type fakeEvents struct {
	events []recordedEvent
	err    error
}

func (f *fakeEvents) Record(_ context.Context, villaSlug, message string) error {
	if f.err != nil {
		return f.err
	}
	f.events = append(f.events, recordedEvent{villaSlug: villaSlug, message: message})
	return nil
}

type fakeBookingReader struct {
	bySlug map[string]string // bookingID -> villaSlug
	err    error
}

func (f fakeBookingReader) GetVillaSlug(_ context.Context, bookingID string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	slug, ok := f.bySlug[bookingID]
	if !ok {
		return "", ErrBookingNotFound
	}
	return slug, nil
}

type fixedClock struct{ t time.Time }

func (f fixedClock) Now() time.Time { return f.t }

func d(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func ptr[T any](v T) *T { return &v }
