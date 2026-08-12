package review

import (
	"context"
	"sort"
	"time"
)

type fakeRepo struct {
	saved   []Review
	meta    map[string]ReviewMeta
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

func (f *fakeRepo) GetMeta(_ context.Context, slug string) (ReviewMeta, error) {
	m, ok := f.meta[slug]
	if !ok {
		return ReviewMeta{VillaSlug: slug}, nil
	}
	return m, nil
}

func (f *fakeRepo) UpsertMeta(_ context.Context, m ReviewMeta) (ReviewMeta, error) {
	if f.meta == nil {
		f.meta = map[string]ReviewMeta{}
	}
	f.meta[m.VillaSlug] = m
	return m, nil
}

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
