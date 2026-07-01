package review

import (
	"context"
	"testing"
)

func newSvc(repo Repository, bookings BookingReader, clock Clock) *Service {
	return NewService(repo, bookings, clock)
}

func TestSubmit_Happy(t *testing.T) {
	repo := &fakeRepo{}
	bookings := fakeBookingReader{bySlug: map[string]string{"booking-1": "casadana"}}
	svc := newSvc(repo, bookings, fixedClock{t: d("2026-08-01")})

	r, err := svc.Submit(context.Background(), SubmitCommand{
		BookingID:  "booking-1",
		AuthorName: "Jane",
		Rating:     5,
		Body:       "Loved it.",
	})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if r.VillaSlug != "casadana" {
		t.Errorf("VillaSlug = %q, want casadana", r.VillaSlug)
	}
	if r.Status != StatusPending {
		t.Errorf("Status = %s, want pending", r.Status)
	}
	if len(repo.saved) != 1 {
		t.Errorf("saved count = %d, want 1", len(repo.saved))
	}
}

func TestSubmit_UnknownBooking(t *testing.T) {
	bookings := fakeBookingReader{bySlug: map[string]string{}}
	svc := newSvc(&fakeRepo{}, bookings, fixedClock{t: d("2026-08-01")})

	_, err := svc.Submit(context.Background(), SubmitCommand{
		BookingID:  "ghost",
		AuthorName: "X", Rating: 5,
	})
	if err != ErrBookingNotFound {
		t.Fatalf("err = %v, want ErrBookingNotFound", err)
	}
}

func TestSubmit_AlreadyReviewed(t *testing.T) {
	repo := &fakeRepo{
		saved: []Review{{ID: "existing", BookingID: "booking-1"}},
	}
	bookings := fakeBookingReader{bySlug: map[string]string{"booking-1": "casadana"}}
	svc := newSvc(repo, bookings, fixedClock{t: d("2026-08-01")})

	_, err := svc.Submit(context.Background(), SubmitCommand{
		BookingID:  "booking-1",
		AuthorName: "X", Rating: 5,
	})
	if err != ErrAlreadyReviewed {
		t.Fatalf("err = %v, want ErrAlreadyReviewed", err)
	}
}

func TestSubmit_BadRating(t *testing.T) {
	bookings := fakeBookingReader{bySlug: map[string]string{"booking-1": "casadana"}}
	svc := newSvc(&fakeRepo{}, bookings, fixedClock{t: d("2026-08-01")})

	_, err := svc.Submit(context.Background(), SubmitCommand{
		BookingID:  "booking-1",
		AuthorName: "X", Rating: 6,
	})
	if err != ErrInvalidPayload {
		t.Fatalf("err = %v, want ErrInvalidPayload", err)
	}
}

func TestListByVilla(t *testing.T) {
	repo := &fakeRepo{
		saved: []Review{
			{ID: "1", VillaSlug: "casadana"},
			{ID: "2", VillaSlug: "casacasay"},
			{ID: "3", VillaSlug: "casadana"},
		},
	}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})

	out, err := svc.ListByVilla(context.Background(), "casadana")
	if err != nil {
		t.Fatalf("ListByVilla: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("len = %d, want 2", len(out))
	}
}

func TestDelete_NotFound(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	if err := svc.Delete(context.Background(), "nope"); err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
