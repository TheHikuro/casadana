package booking

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func newRouter(svc *Service, requireAuth func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	Mount(r, svc, requireAuth)
	return r
}

// noopAuth lets tests that aren't about auth exercise the admin-only routes
// as if a valid session were already present.
func noopAuth(next http.Handler) http.Handler { return next }

// denyAllAuth simulates a middleware that rejects every request — used to
// prove requireAuth is actually wired onto the routes that need it (and not
// onto the ones that shouldn't be gated).
func denyAllAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
}

func TestPostBooking_Created(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, &fakeMailer{},
		fakeAllowlist{allowed: map[string]bool{"casadana": true}},
		d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	body := `{"villa_slug":"casadana","guest_name":"Jane","guest_email":"jane@example.com","guest_phone":"+33123","check_in":"2026-07-01","check_out":"2026-07-08","adults":2,"children":0,"message":"hi"}`
	resp, err := http.Post(srv.URL+"/api/bookings", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	var out bookingResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Status != "pending" {
		t.Errorf("status = %q, want pending", out.Status)
	}
}

func TestPostBooking_ValidationError(t *testing.T) {
	svc := newSvc(&fakeRepo{}, &fakeMailer{},
		fakeAllowlist{allowed: map[string]bool{"casadana": true}},
		d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	body := bytes.NewBufferString(`{"villa_slug":"casadana"}`)
	resp, err := http.Post(srv.URL+"/api/bookings", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestPostBooking_DatesConflict(t *testing.T) {
	repo := &fakeRepo{overlapping: []Booking{{ID: "x"}}}
	svc := newSvc(repo, &fakeMailer{},
		fakeAllowlist{allowed: map[string]bool{"casadana": true}},
		d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	body := `{"villa_slug":"casadana","guest_name":"Jane","guest_email":"jane@example.com","guest_phone":"+33","check_in":"2026-07-01","check_out":"2026-07-08","adults":1,"children":0}`
	resp, err := http.Post(srv.URL+"/api/bookings", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}
}

func TestGetAvailability_Empty(t *testing.T) {
	svc := newSvc(&fakeRepo{}, &fakeMailer{},
		fakeAllowlist{allowed: map[string]bool{"casadana": true}},
		d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/villas/casadana/availability?from=2026-07-01&to=2026-08-01")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestGetAvailability_SeparatesPendingFromBooked(t *testing.T) {
	repo := &fakeRepo{
		bookedRanges:  []DateRange{{CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08")}},
		pendingRanges: []DateRange{{CheckIn: d("2026-07-10"), CheckOut: d("2026-07-12")}},
	}
	svc := newSvc(repo, &fakeMailer{},
		fakeAllowlist{allowed: map[string]bool{"casadana": true}},
		d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/villas/casadana/availability?from=2026-07-01&to=2026-08-01")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got availabilityResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.BookedRanges) != 1 || got.BookedRanges[0].CheckIn != "2026-07-01" {
		t.Errorf("BookedRanges = %+v, want the confirmed range only", got.BookedRanges)
	}
	if len(got.PendingRanges) != 1 || got.PendingRanges[0].CheckIn != "2026-07-10" {
		t.Errorf("PendingRanges = %+v, want the pending range only", got.PendingRanges)
	}
}

func TestListBookings_PaginatedResponse(t *testing.T) {
	repo := &fakeRepo{
		saved: []Booking{
			{ID: "1", Status: StatusPending, GuestEmail: "a@x.com", GuestName: "A", VillaSlug: "casadana", CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08")},
			{ID: "2", Status: StatusApproved, GuestEmail: "b@x.com", GuestName: "B", VillaSlug: "casadana", CheckIn: d("2026-08-01"), CheckOut: d("2026-08-08")},
		},
	}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/bookings?page=1&limit=10")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out listBookingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Total != 2 || out.Page != 1 || out.Limit != 10 {
		t.Errorf("page/limit/total = %d/%d/%d, want 1/10/2", out.Page, out.Limit, out.Total)
	}
	if len(out.Bookings) != 2 {
		t.Errorf("bookings len = %d, want 2", len(out.Bookings))
	}
}

func TestListBookings_BadStatus(t *testing.T) {
	svc := newSvc(&fakeRepo{}, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/bookings?status=invalid")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestDeleteBooking_NoContent(t *testing.T) {
	repo := &fakeRepo{saved: []Booking{{ID: "abc-123"}}}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/bookings/abc-123", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
}

func TestDeleteBooking_NotFound(t *testing.T) {
	svc := newSvc(&fakeRepo{}, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/bookings/nope", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestListBookings_RequiresAuth(t *testing.T) {
	svc := newSvc(&fakeRepo{}, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, denyAllAuth))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/bookings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestPatchBooking_RequiresAuth(t *testing.T) {
	svc := newSvc(&fakeRepo{}, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, denyAllAuth))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPatch, srv.URL+"/api/bookings/abc", strings.NewReader(`{"status":"approved"}`))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestDeleteBooking_RequiresAuth(t *testing.T) {
	svc := newSvc(&fakeRepo{}, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, denyAllAuth))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/bookings/abc", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestPostBooking_NotGatedByAuth(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, denyAllAuth))
	defer srv.Close()

	body := `{"villa_slug":"casadana","guest_name":"Jane","guest_email":"jane@example.com","guest_phone":"+33123","check_in":"2026-07-01","check_out":"2026-07-08","adults":2,"children":0,"message":"hi"}`
	resp, err := http.Post(srv.URL+"/api/bookings", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (POST must not be auth-gated even when requireAuth denies everything)", resp.StatusCode)
	}
}

func TestGetAvailability_NotGatedByAuth(t *testing.T) {
	svc := newSvc(&fakeRepo{}, &fakeMailer{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, denyAllAuth))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/villas/casadana/availability?from=2026-07-01&to=2026-08-01")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestPostBooking_DefaultsSourceToDirect(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	body := `{"villa_slug":"casadana","guest_name":"Jane","guest_email":"jane@example.com","guest_phone":"+33123","check_in":"2026-07-01","check_out":"2026-07-08","adults":2,"children":0}`
	resp, err := http.Post(srv.URL+"/api/bookings", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out bookingResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Source != "direct" {
		t.Errorf("Source = %q, want direct", out.Source)
	}
}

func TestListBookings_FilterByVillaSlug(t *testing.T) {
	repo := &fakeRepo{
		saved: []Booking{
			{ID: "1", VillaSlug: "casadana", GuestName: "A", GuestEmail: "a@x.com", CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08")},
			{ID: "2", VillaSlug: "casacasay", GuestName: "B", GuestEmail: "b@x.com", CheckIn: d("2026-08-01"), CheckOut: d("2026-08-08")},
		},
	}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{allowed: map[string]bool{"casadana": true, "casacasay": true}}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/bookings?villa_slug=casadana")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out listBookingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Total != 1 || len(out.Bookings) != 1 || out.Bookings[0].VillaSlug != "casadana" {
		t.Errorf("out = %+v, want exactly the casadana booking", out)
	}
}
