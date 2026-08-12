package booking

import (
	"context"
	"errors"
	"time"
)

type fakeRepo struct {
	saved         []Booking
	overlapping   []Booking
	bookedRanges  []DateRange
	pendingRanges []DateRange
	saveErr       error
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
func (f *fakeRepo) FindOverlappingConfirmed(_ context.Context, villaSlug string, from, to time.Time, excludeID string) ([]Booking, error) {
	var out []Booking
	for _, b := range f.saved {
		if b.ID == excludeID || b.VillaSlug != villaSlug {
			continue
		}
		if b.Status != StatusApproved && b.Status != StatusPaid {
			continue
		}
		if b.CheckIn.Before(to) && b.CheckOut.After(from) {
			out = append(out, b)
		}
	}
	return out, nil
}
func (f *fakeRepo) BookedRanges(_ context.Context, _ string, _, _ time.Time) ([]DateRange, error) {
	return f.bookedRanges, nil
}
func (f *fakeRepo) PendingRanges(_ context.Context, _ string, _, _ time.Time) ([]DateRange, error) {
	return f.pendingRanges, nil
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

func (f *fakeRepo) List(_ context.Context, villaSlug *string, status *Status, limit, offset int) ([]Booking, error) {
	filtered := make([]Booking, 0, len(f.saved))
	for _, b := range f.saved {
		if villaSlug != nil && b.VillaSlug != *villaSlug {
			continue
		}
		if status != nil && b.Status != *status {
			continue
		}
		filtered = append(filtered, b)
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

func (f *fakeRepo) Count(_ context.Context, villaSlug *string, status *Status) (int, error) {
	n := 0
	for _, b := range f.saved {
		if villaSlug != nil && b.VillaSlug != *villaSlug {
			continue
		}
		if status != nil && b.Status != *status {
			continue
		}
		n++
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
	received     []Booking
	ownerNotices []Booking
	approved     []Booking
	rejected     []Booking
	cancelled    []Booking

	receivedErr    error
	ownerNoticeErr error
	statusErr      error
}

func (f *fakeMailer) SendRequestReceived(_ context.Context, b *Booking) error {
	if f.receivedErr != nil {
		return f.receivedErr
	}
	f.received = append(f.received, *b)
	return nil
}
func (f *fakeMailer) SendOwnerNewRequest(_ context.Context, b *Booking) error {
	if f.ownerNoticeErr != nil {
		return f.ownerNoticeErr
	}
	f.ownerNotices = append(f.ownerNotices, *b)
	return nil
}
func (f *fakeMailer) SendApproved(_ context.Context, b *Booking) error {
	if f.statusErr != nil {
		return f.statusErr
	}
	f.approved = append(f.approved, *b)
	return nil
}
func (f *fakeMailer) SendRejected(_ context.Context, b *Booking) error {
	if f.statusErr != nil {
		return f.statusErr
	}
	f.rejected = append(f.rejected, *b)
	return nil
}
func (f *fakeMailer) SendCancelled(_ context.Context, b *Booking) error {
	if f.statusErr != nil {
		return f.statusErr
	}
	f.cancelled = append(f.cancelled, *b)
	return nil
}

type fixedClock struct{ t time.Time }

func (f fixedClock) Now() time.Time { return f.t }

type fakeAllowlist struct{ allowed map[string]bool }

func (f fakeAllowlist) IsKnown(slug string) bool { return f.allowed[slug] }
