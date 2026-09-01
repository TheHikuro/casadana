package review

import (
	"context"
	"errors"
	"testing"
)

func newSvc(repo Repository, bookings BookingReader, clock Clock) *Service {
	return NewService(repo, bookings, knownVillas(), clock, nil)
}

func newSvcWithAllowlist(repo Repository, allow VillaAllowlist, clock Clock) *Service {
	return NewService(repo, fakeBookingReader{}, allow, clock, nil)
}

func newSvcWithEvents(repo Repository, bookings BookingReader, clock Clock, events EventRecorder) *Service {
	return NewService(repo, bookings, knownVillas(), clock, events)
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

// Guest submission still demands a real booking: the admin path is the only
// way to create a review without one.
func TestSubmit_StillRequiresBooking(t *testing.T) {
	_, err := NewReview(NewReviewInput{VillaSlug: "casadana", AuthorName: "X", Rating: 5})
	if err != ErrInvalidPayload {
		t.Fatalf("err = %v, want ErrInvalidPayload", err)
	}
}

func TestListByVilla_ApprovedOnly(t *testing.T) {
	repo := &fakeRepo{
		saved: []Review{
			{ID: "1", VillaSlug: "casadana", Status: StatusApproved},
			{ID: "2", VillaSlug: "casadana", Status: StatusPending},
			{ID: "3", VillaSlug: "casadana", Status: StatusRejected},
			{ID: "4", VillaSlug: "casacasay", Status: StatusApproved},
		},
	}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})

	out, err := svc.ListByVilla(context.Background(), "casadana")
	if err != nil {
		t.Fatalf("ListByVilla: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len = %d, want 1 (approved only)", len(out))
	}
	if out[0].ID != "1" {
		t.Errorf("ID = %q, want 1", out[0].ID)
	}
}

func TestListForAdmin(t *testing.T) {
	repo := &fakeRepo{
		saved: []Review{
			{ID: "1", VillaSlug: "casadana", Status: StatusApproved},
			{ID: "2", VillaSlug: "casadana", Status: StatusPending},
			{ID: "3", VillaSlug: "casacasay", Status: StatusPending},
		},
	}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})

	tests := []struct {
		name   string
		status *Status
		want   int
	}{
		{name: "all statuses", status: nil, want: 2},
		{name: "pending only", status: ptr(StatusPending), want: 1},
		{name: "rejected only", status: ptr(StatusRejected), want: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := svc.ListForAdmin(context.Background(), "casadana", tc.status)
			if err != nil {
				t.Fatalf("ListForAdmin: %v", err)
			}
			if len(out) != tc.want {
				t.Errorf("len = %d, want %d", len(out), tc.want)
			}
		})
	}
}

func TestListForAdmin_BadStatus(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	if _, err := svc.ListForAdmin(context.Background(), "casadana", ptr(Status("archived"))); err != ErrInvalidPayload {
		t.Fatalf("err = %v, want ErrInvalidPayload", err)
	}
}

func TestListForAdmin_FeaturedFirst(t *testing.T) {
	repo := &fakeRepo{
		saved: []Review{
			{ID: "plain", VillaSlug: "casadana", Status: StatusApproved},
			{ID: "star", VillaSlug: "casadana", Status: StatusApproved, Featured: true},
		},
	}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})

	out, err := svc.ListForAdmin(context.Background(), "casadana", nil)
	if err != nil {
		t.Fatalf("ListForAdmin: %v", err)
	}
	if out[0].ID != "star" {
		t.Errorf("first = %q, want star", out[0].ID)
	}
}

func TestCreateByAdmin_DefaultsToApprovedWithNoBooking(t *testing.T) {
	repo := &fakeRepo{}
	events := &fakeEvents{}
	svc := newSvcWithEvents(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")}, events)

	r, err := svc.CreateByAdmin(context.Background(), CreateByAdminCommand{
		VillaSlug:  "casadana",
		AuthorName: "Marta Ruiz",
		Rating:     5,
		Body:       "Transcribed from Airbnb.",
		Source:     "airbnb",
	})
	if err != nil {
		t.Fatalf("CreateByAdmin: %v", err)
	}
	if r.BookingID != "" {
		t.Errorf("BookingID = %q, want empty", r.BookingID)
	}
	if r.Status != StatusApproved {
		t.Errorf("Status = %s, want approved", r.Status)
	}
	if r.Source != "airbnb" {
		t.Errorf("Source = %q, want airbnb", r.Source)
	}
	if len(events.events) != 1 || events.events[0].villaSlug != "casadana" {
		t.Fatalf("events = %+v, want one for casadana", events.events)
	}
}

func TestCreateByAdmin_Invalid(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})

	tests := []struct {
		name string
		cmd  CreateByAdminCommand
	}{
		{name: "no villa", cmd: CreateByAdminCommand{AuthorName: "X", Rating: 5}},
		{name: "no author", cmd: CreateByAdminCommand{VillaSlug: "casadana", Rating: 5}},
		{name: "rating too high", cmd: CreateByAdminCommand{VillaSlug: "casadana", AuthorName: "X", Rating: 6}},
		{name: "rating too low", cmd: CreateByAdminCommand{VillaSlug: "casadana", AuthorName: "X", Rating: 0}},
		{name: "unknown status", cmd: CreateByAdminCommand{VillaSlug: "casadana", AuthorName: "X", Rating: 5, Status: "archived"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.CreateByAdmin(context.Background(), tc.cmd); err != ErrInvalidPayload {
				t.Fatalf("err = %v, want ErrInvalidPayload", err)
			}
		})
	}
}

func TestCreateByAdmin_BodyTooLong(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	body := make([]byte, 2001)
	for i := range body {
		body[i] = 'a'
	}
	_, err := svc.CreateByAdmin(context.Background(), CreateByAdminCommand{
		VillaSlug: "casadana", AuthorName: "X", Rating: 5, Body: string(body),
	})
	if err != ErrInvalidPayload {
		t.Fatalf("err = %v, want ErrInvalidPayload", err)
	}
}

func TestUpdateStatus_RecordsEvent(t *testing.T) {
	repo := &fakeRepo{saved: []Review{
		{ID: "r1", VillaSlug: "casadana", AuthorName: "Marta Ruiz", Status: StatusPending},
	}}
	events := &fakeEvents{}
	svc := newSvcWithEvents(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")}, events)

	r, err := svc.UpdateStatus(context.Background(), "r1", StatusApproved)
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	if r.Status != StatusApproved {
		t.Errorf("Status = %s, want approved", r.Status)
	}
	want := "Review by Marta Ruiz set to approved"
	if len(events.events) != 1 || events.events[0].message != want {
		t.Fatalf("events = %+v, want message %q", events.events, want)
	}
}

func TestUpdate_AllFields(t *testing.T) {
	repo := &fakeRepo{saved: []Review{
		{ID: "r1", VillaSlug: "casadana", AuthorName: "Marta Ruiz", Status: StatusPending, Rating: 3},
	}}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})

	r, err := svc.Update(context.Background(), "r1", UpdatePatch{
		Status:   ptr(StatusApproved),
		Featured: ptr(true),
		Meta:     ptr("stay: 5 nights"),
		Source:   ptr("airbnb"),
		Body:     ptr("Edited body."),
		Rating:   ptr(5),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if r.Status != StatusApproved || !r.Featured || r.Meta != "stay: 5 nights" ||
		r.Source != "airbnb" || r.Body != "Edited body." || r.Rating != 5 {
		t.Errorf("unexpected review: %+v", r)
	}
}

func TestUpdate_PartialLeavesRest(t *testing.T) {
	repo := &fakeRepo{saved: []Review{
		{ID: "r1", VillaSlug: "casadana", AuthorName: "Marta Ruiz", Status: StatusApproved, Rating: 4, Body: "Original"},
	}}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})

	r, err := svc.Update(context.Background(), "r1", UpdatePatch{Featured: ptr(true)})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !r.Featured || r.Status != StatusApproved || r.Rating != 4 || r.Body != "Original" {
		t.Errorf("unexpected review: %+v", r)
	}
}

func TestUpdate_Invalid(t *testing.T) {
	repo := &fakeRepo{saved: []Review{{ID: "r1", VillaSlug: "casadana"}}}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})

	tests := []struct {
		name  string
		patch UpdatePatch
	}{
		{name: "unknown status", patch: UpdatePatch{Status: ptr(Status("archived"))}},
		{name: "rating too high", patch: UpdatePatch{Rating: ptr(6)}},
		{name: "rating too low", patch: UpdatePatch{Rating: ptr(0)}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Update(context.Background(), "r1", tc.patch); err != ErrInvalidPayload {
				t.Fatalf("err = %v, want ErrInvalidPayload", err)
			}
		})
	}
}

func TestUpdate_NotFound(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	if _, err := svc.Update(context.Background(), "ghost", UpdatePatch{Status: ptr(StatusApproved)}); err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

// A failing activity log is a logging problem, not a moderation problem: the
// row is already updated, so the request must still succeed.
func TestUpdate_EventErrorDoesNotFail(t *testing.T) {
	repo := &fakeRepo{saved: []Review{{ID: "r1", VillaSlug: "casadana", AuthorName: "Marta Ruiz"}}}
	events := &fakeEvents{err: errors.New("audit down")}
	svc := newSvcWithEvents(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")}, events)

	if _, err := svc.UpdateStatus(context.Background(), "r1", StatusApproved); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
}

func TestUpdate_NilRecorderIsNoOp(t *testing.T) {
	repo := &fakeRepo{saved: []Review{{ID: "r1", VillaSlug: "casadana", AuthorName: "Marta Ruiz"}}}
	svc := newSvcWithEvents(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")}, nil)

	if _, err := svc.UpdateStatus(context.Background(), "r1", StatusRejected); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
}

func TestPatchMessage(t *testing.T) {
	r := &Review{AuthorName: "Marta Ruiz"}
	tests := []struct {
		name  string
		patch UpdatePatch
		want  string
	}{
		{name: "status wins", patch: UpdatePatch{Status: ptr(StatusRejected)}, want: "Review by Marta Ruiz set to rejected"},
		{name: "featured", patch: UpdatePatch{Featured: ptr(true)}, want: "Review by Marta Ruiz featured"},
		{name: "unfeatured", patch: UpdatePatch{Featured: ptr(false)}, want: "Review by Marta Ruiz unfeatured"},
		{name: "plain edit", patch: UpdatePatch{Body: ptr("x")}, want: "Review by Marta Ruiz edited"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := patchMessage(r, tc.patch); got != tc.want {
				t.Errorf("patchMessage = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDelete_NotFound(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	if err := svc.Delete(context.Background(), "nope"); err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestDelete_RecordsEvent(t *testing.T) {
	repo := &fakeRepo{saved: []Review{{ID: "r1", VillaSlug: "casadana", AuthorName: "Marta Ruiz"}}}
	events := &fakeEvents{}
	svc := newSvcWithEvents(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")}, events)

	if err := svc.Delete(context.Background(), "r1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(repo.saved) != 0 {
		t.Errorf("saved count = %d, want 0", len(repo.saved))
	}
	want := "Review by Marta Ruiz deleted"
	if len(events.events) != 1 || events.events[0].message != want {
		t.Fatalf("events = %+v, want message %q", events.events, want)
	}
}

func TestMeta_NoApprovedReviewsIsZero(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})

	m, err := svc.Meta(context.Background(), "casadana")
	if err != nil {
		t.Fatalf("Meta: %v", err)
	}
	if m.DisplayAvg != 0 || m.DisplayCount != 0 || m.Breakdown != (Breakdown{}) {
		t.Errorf("meta = %+v, want zero values", m)
	}
}

// The published rating is derived, so it can only ever reflect the approved
// reviews the villa actually has.
func TestMeta_AveragesApprovedReviews(t *testing.T) {
	repo := &fakeRepo{saved: []Review{
		{ID: "r1", VillaSlug: "casadana", Status: StatusApproved, Rating: 5,
			Categories: CategoryRatings{Cleanliness: ptr(5.0), Value: ptr(4.0)}},
		{ID: "r2", VillaSlug: "casadana", Status: StatusApproved, Rating: 4,
			Categories: CategoryRatings{Cleanliness: ptr(4.5)}},
		{ID: "r3", VillaSlug: "casadana", Status: StatusPending, Rating: 1,
			Categories: CategoryRatings{Cleanliness: ptr(1.0)}},
	}}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})

	m, err := svc.Meta(context.Background(), "casadana")
	if err != nil {
		t.Fatalf("Meta: %v", err)
	}
	if m.DisplayCount != 2 || m.DisplayAvg != 4.5 {
		t.Errorf("meta = %+v, want avg 4.5 over 2 approved reviews", m)
	}
	if m.Breakdown.Cleanliness == nil || *m.Breakdown.Cleanliness != 4.75 {
		t.Errorf("cleanliness = %v, want 4.75", m.Breakdown.Cleanliness)
	}
	// One review scored value, so it stands alone as the average.
	if m.Breakdown.Value == nil || *m.Breakdown.Value != 4 {
		t.Errorf("value = %v, want 4", m.Breakdown.Value)
	}
	// No approved review scored comfort at all.
	if m.Breakdown.Comfort != nil {
		t.Errorf("comfort = %v, want nil", *m.Breakdown.Comfort)
	}
}

// A category score outside 1..5 is rejected before it can skew an average.
func TestCreateByAdmin_RejectsOutOfRangeCategory(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})

	for _, tc := range []struct {
		name string
		cats CategoryRatings
	}{
		{name: "above five", cats: CategoryRatings{Host: ptr(6.0)}},
		{name: "below one", cats: CategoryRatings{Value: ptr(0.5)}},
		{name: "negative", cats: CategoryRatings{Comfort: ptr(-1.0)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CreateByAdmin(context.Background(), CreateByAdminCommand{
				VillaSlug: "casadana", AuthorName: "Ana", Rating: 5, Categories: tc.cats,
			})
			if err != ErrInvalidPayload {
				t.Fatalf("err = %v, want ErrInvalidPayload", err)
			}
		})
	}
}

// A visitor leaving a review on a villa page has no booking to point at, so
// the villa comes from the URL and the review carries none.
func TestSubmitPublic_LandsPendingWithNoBooking(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})

	r, err := svc.SubmitPublic(context.Background(), SubmitPublicCommand{
		VillaSlug:  "casadana",
		AuthorName: "  Ana Ruiz  ",
		Rating:     5,
		Body:       "  Une semaine parfaite.  ",
		Categories: CategoryRatings{Cleanliness: ptr(5.0), Host: ptr(4.0)},
	})
	if err != nil {
		t.Fatalf("SubmitPublic: %v", err)
	}
	if r.Status != StatusPending {
		t.Errorf("Status = %s, want pending — nothing a stranger types is published unmoderated", r.Status)
	}
	if r.BookingID != "" {
		t.Errorf("BookingID = %q, want empty", r.BookingID)
	}
	if r.Source != SourceWebsite {
		t.Errorf("Source = %q, want %q", r.Source, SourceWebsite)
	}
	if r.Featured {
		t.Error("Featured = true, want false")
	}
	if r.AuthorName != "Ana Ruiz" || r.Body != "Une semaine parfaite." {
		t.Errorf("name/body not trimmed: %q / %q", r.AuthorName, r.Body)
	}
	if r.Categories.Cleanliness == nil || *r.Categories.Cleanliness != 5 {
		t.Errorf("cleanliness = %v, want 5", r.Categories.Cleanliness)
	}
	if len(repo.saved) != 1 {
		t.Errorf("saved count = %d, want 1", len(repo.saved))
	}
}

// Both houses take reviews, not just the one.
func TestSubmitPublic_WorksForEveryKnownVilla(t *testing.T) {
	for _, slug := range []string{"casadana", "casacasay"} {
		t.Run(slug, func(t *testing.T) {
			repo := &fakeRepo{}
			svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})

			r, err := svc.SubmitPublic(context.Background(), SubmitPublicCommand{
				VillaSlug: slug, AuthorName: "Ana", Rating: 4, Body: "Lovely",
			})
			if err != nil {
				t.Fatalf("SubmitPublic: %v", err)
			}
			if r.VillaSlug != slug {
				t.Errorf("VillaSlug = %q, want %q", r.VillaSlug, slug)
			}
		})
	}
}

// The slug is whatever sat in the URL, so an unknown one must not create a row
// under a villa that does not exist.
func TestSubmitPublic_UnknownVilla(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvcWithAllowlist(repo, knownVillas(), fixedClock{t: d("2026-08-01")})

	_, err := svc.SubmitPublic(context.Background(), SubmitPublicCommand{
		VillaSlug: "casa-ghost", AuthorName: "Ana", Rating: 5, Body: "Nice",
	})
	if err != ErrUnknownVilla {
		t.Fatalf("err = %v, want ErrUnknownVilla", err)
	}
	if len(repo.saved) != 0 {
		t.Errorf("saved count = %d, want 0", len(repo.saved))
	}
}

func TestSubmitPublic_RejectsBadPayload(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})

	for _, tc := range []struct {
		name string
		cmd  SubmitPublicCommand
	}{
		{name: "rating too high", cmd: SubmitPublicCommand{VillaSlug: "casadana", AuthorName: "Ana", Rating: 6, Body: "Nice"}},
		{name: "rating zero", cmd: SubmitPublicCommand{VillaSlug: "casadana", AuthorName: "Ana", Rating: 0, Body: "Nice"}},
		{name: "blank name", cmd: SubmitPublicCommand{VillaSlug: "casadana", AuthorName: "   ", Rating: 5, Body: "Nice"}},
		{name: "blank body", cmd: SubmitPublicCommand{VillaSlug: "casadana", AuthorName: "Ana", Rating: 5, Body: "   "}},
		{name: "category out of range", cmd: SubmitPublicCommand{VillaSlug: "casadana", AuthorName: "Ana", Rating: 5, Body: "Nice", Categories: CategoryRatings{Value: ptr(9.0)}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.SubmitPublic(context.Background(), tc.cmd); err != ErrInvalidPayload {
				t.Fatalf("err = %v, want ErrInvalidPayload", err)
			}
		})
	}
	if len(repo.saved) != 0 {
		t.Errorf("saved count = %d, want 0", len(repo.saved))
	}
}

// The whole point of landing pending: the figures a guest reads must not budge
// until an admin has approved the submission.
func TestSubmitPublic_DoesNotMovePublishedFigures(t *testing.T) {
	repo := &fakeRepo{saved: []Review{
		{ID: "r1", VillaSlug: "casadana", Status: StatusApproved, Rating: 5},
	}}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})

	submitted, err := svc.SubmitPublic(context.Background(), SubmitPublicCommand{
		VillaSlug: "casadana", AuthorName: "Troll", Rating: 1, Body: "Awful",
	})
	if err != nil {
		t.Fatalf("SubmitPublic: %v", err)
	}

	meta, err := svc.Meta(context.Background(), "casadana")
	if err != nil {
		t.Fatalf("Meta: %v", err)
	}
	if meta.DisplayAvg != 5 || meta.DisplayCount != 1 {
		t.Errorf("meta = %+v, want avg 5 over 1 review (the pending one excluded)", meta)
	}

	published, err := svc.ListByVilla(context.Background(), "casadana")
	if err != nil {
		t.Fatalf("ListByVilla: %v", err)
	}
	for _, r := range published {
		if r.ID == submitted.ID {
			t.Error("the pending submission leaked into the public listing")
		}
	}

	// Approving it is what lets it count.
	if _, err := svc.UpdateStatus(context.Background(), submitted.ID, StatusApproved); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	after, err := svc.Meta(context.Background(), "casadana")
	if err != nil {
		t.Fatalf("Meta: %v", err)
	}
	if after.DisplayAvg != 3 || after.DisplayCount != 2 {
		t.Errorf("meta after approval = %+v, want avg 3 over 2 reviews", after)
	}
}

func TestSubmitPublic_RecordsEvent(t *testing.T) {
	events := &fakeEvents{}
	svc := newSvcWithEvents(&fakeRepo{}, fakeBookingReader{}, fixedClock{t: d("2026-08-01")}, events)

	if _, err := svc.SubmitPublic(context.Background(), SubmitPublicCommand{
		VillaSlug: "casadana", AuthorName: "Ana", Rating: 5, Body: "Nice",
	}); err != nil {
		t.Fatalf("SubmitPublic: %v", err)
	}
	if len(events.events) != 1 || events.events[0].villaSlug != "casadana" {
		t.Fatalf("events = %+v, want one line for casadana", events.events)
	}
}
