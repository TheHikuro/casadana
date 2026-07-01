package booking

import (
	"context"
	"testing"
	"time"
)

func newSvc(repo Repository, mailer Mailer, allow VillaAllowlist, now time.Time) *Service {
	return NewService(repo, mailer, allow, fixedClock{t: now})
}

func TestCreate_Happy(t *testing.T) {
	repo := &fakeRepo{}
	mailer := &fakeMailer{}
	allow := fakeAllowlist{allowed: map[string]bool{"casadana": true}}
	svc := newSvc(repo, mailer, allow, d("2026-05-12"))

	b, err := svc.Create(context.Background(), CreateCommand{
		VillaSlug:  "casadana",
		GuestName:  "Jane",
		GuestEmail: "jane@example.com",
		GuestPhone: "+33",
		CheckIn:    d("2026-07-01"),
		CheckOut:   d("2026-07-08"),
		Adults:     2,
		Children:   0,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got, want := len(repo.saved), 1; got != want {
		t.Errorf("saved count = %d, want %d", got, want)
	}
	if got, want := len(mailer.confirmations), 1; got != want {
		t.Errorf("confirmations = %d, want %d", got, want)
	}
	if got, want := len(mailer.adminNotices), 1; got != want {
		t.Errorf("admin notices = %d, want %d", got, want)
	}
	if b.Status != StatusPending {
		t.Errorf("status = %s, want pending", b.Status)
	}
}

func TestCreate_UnknownVilla(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{allowed: map[string]bool{}}, d("2026-05-12"))

	_, err := svc.Create(context.Background(), CreateCommand{
		VillaSlug: "ghost-villa",
		GuestName: "X", GuestEmail: "x@example.com",
		CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08"),
		Adults: 1,
	})
	if err == nil || !isErr(err, ErrUnknownVilla) {
		t.Fatalf("err = %v, want ErrUnknownVilla", err)
	}
	if len(repo.saved) != 0 {
		t.Error("repo should not have been written")
	}
}

func TestCreate_DatesConflict(t *testing.T) {
	repo := &fakeRepo{overlapping: []Booking{{ID: "x"}}}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}}, d("2026-05-12"))

	_, err := svc.Create(context.Background(), CreateCommand{
		VillaSlug: "casadana",
		GuestName: "Jane", GuestEmail: "jane@example.com",
		CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08"),
		Adults: 1,
	})
	if err == nil || !isErr(err, ErrDatesConflict) {
		t.Fatalf("err = %v, want ErrDatesConflict", err)
	}
	if len(repo.saved) != 0 {
		t.Error("repo should not have been written")
	}
}

func TestCreate_MailerFailure_DoesNotFailBooking(t *testing.T) {
	repo := &fakeRepo{}
	mailer := &fakeMailer{confirmErr: errBoom}
	svc := newSvc(repo, mailer, fakeAllowlist{allowed: map[string]bool{"casadana": true}}, d("2026-05-12"))

	_, err := svc.Create(context.Background(), CreateCommand{
		VillaSlug: "casadana",
		GuestName: "Jane", GuestEmail: "jane@example.com",
		CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08"),
		Adults: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.saved) != 1 {
		t.Error("booking should have been persisted despite mailer failure")
	}
}

func TestList_Pagination(t *testing.T) {
	repo := &fakeRepo{}
	allow := fakeAllowlist{allowed: map[string]bool{"casadana": true}}
	svc := newSvc(repo, &fakeMailer{}, allow, d("2026-05-12"))

	for i := 0; i < 3; i++ {
		_, err := svc.Create(context.Background(), CreateCommand{
			VillaSlug:  "casadana",
			GuestName:  "Jane",
			GuestEmail: "jane@example.com",
			GuestPhone: "+33",
			CheckIn:    d("2026-07-01").AddDate(0, 0, i*10),
			CheckOut:   d("2026-07-08").AddDate(0, 0, i*10),
			Adults:     2,
		})
		if err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}

	bookings, total, err := svc.List(context.Background(), nil, 1, 2)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(bookings) != 2 {
		t.Errorf("len = %d, want 2 (limit=2)", len(bookings))
	}
}

func TestList_StatusFilter(t *testing.T) {
	repo := &fakeRepo{
		saved: []Booking{
			{ID: "1", Status: StatusPending},
			{ID: "2", Status: StatusApproved},
			{ID: "3", Status: StatusPending},
		},
	}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))

	pending := StatusPending
	bookings, total, err := svc.List(context.Background(), &pending, 1, 50)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(bookings) != 2 {
		t.Errorf("len = %d, want 2", len(bookings))
	}
}

func TestList_Clamps(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))

	cases := []struct {
		page, limit int
	}{
		{0, 20},
		{1, 0},
		{1, 200},
		{3, 25},
	}
	for _, c := range cases {
		_, _, err := svc.List(context.Background(), nil, c.page, c.limit)
		if err != nil {
			t.Fatalf("page=%d limit=%d: %v", c.page, c.limit, err)
		}
	}
}

func TestDelete_NotFound(t *testing.T) {
	svc := newSvc(&fakeRepo{}, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))
	if err := svc.Delete(context.Background(), "nonexistent-id"); err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestDelete_Happy(t *testing.T) {
	repo := &fakeRepo{saved: []Booking{{ID: "abc-123"}}}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))

	if err := svc.Delete(context.Background(), "abc-123"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(repo.saved) != 0 {
		t.Errorf("repo not emptied: %d remaining", len(repo.saved))
	}
}

// helpers
var errBoom = simpleErr("boom")

type simpleErr string

func (s simpleErr) Error() string { return string(s) }

func isErr(err, target error) bool {
	for e := err; e != nil; {
		if e == target {
			return true
		}
		type unwrap interface{ Unwrap() error }
		u, ok := e.(unwrap)
		if !ok {
			return false
		}
		e = u.Unwrap()
	}
	return false
}
