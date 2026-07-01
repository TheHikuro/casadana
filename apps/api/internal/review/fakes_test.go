package review

import (
	"context"
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
		if existing.BookingID == r.BookingID {
			return ErrAlreadyReviewed
		}
	}
	f.saved = append(f.saved, *r)
	return nil
}

func (f *fakeRepo) ListByVillaSlug(_ context.Context, slug string) ([]Review, error) {
	out := []Review{}
	for _, r := range f.saved {
		if r.VillaSlug == slug {
			out = append(out, r)
		}
	}
	return out, nil
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
