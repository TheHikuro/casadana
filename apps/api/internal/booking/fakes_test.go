package booking

import (
	"context"
	"errors"
	"time"
)

type fakeRepo struct {
	saved        []Booking
	overlapping  []Booking
	bookedRanges []DateRange
	saveErr      error
}

func (f *fakeRepo) Save(_ context.Context, b *Booking) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = append(f.saved, *b)
	return nil
}
func (f *fakeRepo) FindOverlapping(_ context.Context, _ string, _, _ time.Time) ([]Booking, error) {
	return f.overlapping, nil
}
func (f *fakeRepo) BookedRanges(_ context.Context, _ string, _, _ time.Time) ([]DateRange, error) {
	return f.bookedRanges, nil
}
func (f *fakeRepo) Get(_ context.Context, id string) (*Booking, error) {
	for i := range f.saved {
		if f.saved[i].ID == id {
			b := f.saved[i]
			return &b, nil
		}
	}
	return nil, errors.New("not found")
}
func (f *fakeRepo) UpdateStatus(_ context.Context, id string, status Status, updatedAt time.Time) error {
	for i := range f.saved {
		if f.saved[i].ID == id {
			f.saved[i].Status = status
			f.saved[i].UpdatedAt = updatedAt
			return nil
		}
	}
	return errors.New("not found")
}

func (f *fakeRepo) List(_ context.Context, status *Status, limit, offset int) ([]Booking, error) {
	filtered := f.saved
	if status != nil {
		filtered = nil
		for _, b := range f.saved {
			if b.Status == *status {
				filtered = append(filtered, b)
			}
		}
	}
	if offset >= len(filtered) {
		return []Booking{}, nil
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], nil
}

func (f *fakeRepo) Count(_ context.Context, status *Status) (int, error) {
	if status == nil {
		return len(f.saved), nil
	}
	n := 0
	for _, b := range f.saved {
		if b.Status == *status {
			n++
		}
	}
	return n, nil
}

func (f *fakeRepo) Delete(_ context.Context, id string) error {
	for i, b := range f.saved {
		if b.ID == id {
			f.saved = append(f.saved[:i], f.saved[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

type fakeMailer struct {
	confirmations  []Booking
	adminNotices   []Booking
	confirmErr     error
	adminNoticeErr error
}

func (f *fakeMailer) SendBookingConfirmation(_ context.Context, b *Booking) error {
	if f.confirmErr != nil {
		return f.confirmErr
	}
	f.confirmations = append(f.confirmations, *b)
	return nil
}
func (f *fakeMailer) SendAdminNotification(_ context.Context, b *Booking) error {
	if f.adminNoticeErr != nil {
		return f.adminNoticeErr
	}
	f.adminNotices = append(f.adminNotices, *b)
	return nil
}

type fixedClock struct{ t time.Time }

func (f fixedClock) Now() time.Time { return f.t }

type fakeAllowlist struct{ allowed map[string]bool }

func (f fakeAllowlist) IsKnown(slug string) bool { return f.allowed[slug] }
