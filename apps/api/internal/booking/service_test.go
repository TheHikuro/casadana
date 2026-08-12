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
	if got, want := len(mailer.received), 1; got != want {
		t.Errorf("guest request-received emails = %d, want %d", got, want)
	}
	if got, want := len(mailer.ownerNotices), 1; got != want {
		t.Errorf("owner notices = %d, want %d", got, want)
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
	mailer := &fakeMailer{receivedErr: errBoom}
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

	bookings, total, err := svc.List(context.Background(), nil, nil, 1, 2)
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
	bookings, total, err := svc.List(context.Background(), nil, &pending, 1, 50)
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
		_, _, err := svc.List(context.Background(), nil, nil, c.page, c.limit)
		if err != nil {
			t.Fatalf("page=%d limit=%d: %v", c.page, c.limit, err)
		}
	}
}

func TestList_FilterByVillaSlug(t *testing.T) {
	repo := &fakeRepo{
		saved: []Booking{
			{ID: "1", VillaSlug: "casadana", Status: StatusPending},
			{ID: "2", VillaSlug: "casacasay", Status: StatusPending},
			{ID: "3", VillaSlug: "casadana", Status: StatusApproved},
		},
	}
	allow := fakeAllowlist{allowed: map[string]bool{"casadana": true, "casacasay": true}}
	svc := newSvc(repo, &fakeMailer{}, allow, d("2026-05-12"))

	slug := "casadana"
	bookings, total, err := svc.List(context.Background(), &slug, nil, 1, 50)
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

func TestList_UnknownVillaSlug(t *testing.T) {
	svc := newSvc(&fakeRepo{}, &fakeMailer{}, fakeAllowlist{allowed: map[string]bool{}}, d("2026-05-12"))

	slug := "ghost-villa"
	_, _, err := svc.List(context.Background(), &slug, nil, 1, 20)
	if err == nil || !isErr(err, ErrUnknownVilla) {
		t.Fatalf("err = %v, want ErrUnknownVilla", err)
	}
}

func TestAvailability_SeparatesPendingFromBooked(t *testing.T) {
	repo := &fakeRepo{
		bookedRanges:  []DateRange{{CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08")}},
		pendingRanges: []DateRange{{CheckIn: d("2026-07-10"), CheckOut: d("2026-07-12")}},
	}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}}, d("2026-05-12"))

	avail, err := svc.Availability(context.Background(), "casadana", d("2026-07-01"), d("2026-08-01"))
	if err != nil {
		t.Fatalf("Availability: %v", err)
	}
	if len(avail.Booked) != 1 || !avail.Booked[0].CheckIn.Equal(d("2026-07-01")) {
		t.Errorf("Booked = %+v, want the confirmed range", avail.Booked)
	}
	if len(avail.Pending) != 1 || !avail.Pending[0].CheckIn.Equal(d("2026-07-10")) {
		t.Errorf("Pending = %+v, want the pending range", avail.Pending)
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

func TestTransitionStatus_ApproveConflictingDates_Fails(t *testing.T) {
	repo := &fakeRepo{
		saved: []Booking{
			{
				ID: "a", VillaSlug: "casadana", Status: StatusApproved,
				CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08"),
			},
			{
				ID: "b", VillaSlug: "casadana", Status: StatusPending,
				CheckIn: d("2026-07-05"), CheckOut: d("2026-07-10"),
			},
		},
	}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))

	_, err := svc.TransitionStatus(context.Background(), "b", StatusApproved)
	if err == nil || !isErr(err, ErrDatesConflict) {
		t.Fatalf("err = %v, want ErrDatesConflict", err)
	}
	got, _ := repo.Get(context.Background(), "b")
	if got.Status != StatusPending {
		t.Errorf("status = %s, want unchanged pending", got.Status)
	}
}

func TestTransitionStatus_ApproveNonConflictingDates_Happy(t *testing.T) {
	repo := &fakeRepo{
		saved: []Booking{
			{
				ID: "a", VillaSlug: "casadana", Status: StatusApproved,
				CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08"),
			},
			{
				ID: "b", VillaSlug: "casadana", Status: StatusPending,
				CheckIn: d("2026-07-08"), CheckOut: d("2026-07-12"),
			},
		},
	}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))

	b, err := svc.TransitionStatus(context.Background(), "b", StatusApproved)
	if err != nil {
		t.Fatalf("TransitionStatus: %v", err)
	}
	if b.Status != StatusApproved {
		t.Errorf("status = %s, want approved", b.Status)
	}
}

func TestTransitionStatus_ApproveIgnoresRejectedAndCancelled(t *testing.T) {
	repo := &fakeRepo{
		saved: []Booking{
			{
				ID: "a", VillaSlug: "casadana", Status: StatusRejected,
				CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08"),
			},
			{
				ID: "c", VillaSlug: "casadana", Status: StatusCancelled,
				CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08"),
			},
			{
				ID: "b", VillaSlug: "casadana", Status: StatusPending,
				CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08"),
			},
		},
	}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))

	_, err := svc.TransitionStatus(context.Background(), "b", StatusApproved)
	if err != nil {
		t.Fatalf("TransitionStatus: %v, want no conflict against rejected/cancelled bookings", err)
	}
}

// Every transition a guest can see must produce exactly one email to that
// guest, and never one of the other kinds: an approved stay must not also be
// told it was cancelled.
func TestTransitionStatus_EmailsTheGuest(t *testing.T) {
	tests := []struct {
		name    string
		from    Status
		to      Status
		mailbox func(*fakeMailer) []Booking
	}{
		{"approved", StatusPending, StatusApproved, func(m *fakeMailer) []Booking { return m.approved }},
		{"rejected", StatusPending, StatusRejected, func(m *fakeMailer) []Booking { return m.rejected }},
		{"cancelled", StatusPending, StatusCancelled, func(m *fakeMailer) []Booking { return m.cancelled }},
		{"paid stays silent", StatusApproved, StatusPaid, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepo{saved: []Booking{{
				ID: "b", VillaSlug: "casadana", Status: tc.from,
				GuestEmail: "jane@example.com",
				CheckIn:    d("2026-07-01"), CheckOut: d("2026-07-08"),
			}}}
			mailer := &fakeMailer{}
			svc := newSvc(repo, mailer, fakeAllowlist{}, d("2026-05-12"))

			if _, err := svc.TransitionStatus(context.Background(), "b", tc.to); err != nil {
				t.Fatalf("TransitionStatus: %v", err)
			}

			sent := len(mailer.approved) + len(mailer.rejected) + len(mailer.cancelled)
			if tc.mailbox == nil {
				if sent != 0 {
					t.Fatalf("sent %d guest emails, want none for %s", sent, tc.to)
				}
				return
			}
			if got := len(tc.mailbox(mailer)); got != 1 {
				t.Errorf("%s emails = %d, want 1", tc.to, got)
			}
			if sent != 1 {
				t.Errorf("total guest emails = %d, want exactly 1", sent)
			}
		})
	}
}

// A dead mail provider must not make the owners believe an approval failed:
// the status is already persisted by the time the email is attempted.
func TestTransitionStatus_MailerFailure_KeepsTheTransition(t *testing.T) {
	repo := &fakeRepo{saved: []Booking{{
		ID: "b", VillaSlug: "casadana", Status: StatusPending,
		CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08"),
	}}}
	svc := newSvc(repo, &fakeMailer{statusErr: errBoom}, fakeAllowlist{}, d("2026-05-12"))

	b, err := svc.TransitionStatus(context.Background(), "b", StatusApproved)
	if err != nil {
		t.Fatalf("TransitionStatus: %v", err)
	}
	if b.Status != StatusApproved {
		t.Errorf("returned status = %s, want approved", b.Status)
	}
	if repo.saved[0].Status != StatusApproved {
		t.Errorf("persisted status = %s, want approved despite the mail failure", repo.saved[0].Status)
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
