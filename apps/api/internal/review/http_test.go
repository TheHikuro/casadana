package review

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func newRouter(svc *Service) http.Handler {
	r := chi.NewRouter()
	Mount(r, svc)
	return r
}

func TestSubmitReview_Created(t *testing.T) {
	bookingID := "11111111-1111-1111-1111-111111111111"
	repo := &fakeRepo{}
	bookings := fakeBookingReader{bySlug: map[string]string{bookingID: "casadana"}}
	svc := newSvc(repo, bookings, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	body := `{"booking_id":"` + bookingID + `","author_name":"Jane","rating":5,"body":"Great"}`
	resp, err := http.Post(srv.URL+"/api/reviews", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	var out reviewDTO
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Status != "pending" || out.VillaSlug != "casadana" {
		t.Errorf("unexpected response: %+v", out)
	}
}

func TestSubmitReview_BookingNotFound(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeBookingReader{bySlug: map[string]string{}}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	body := `{"booking_id":"11111111-1111-1111-1111-111111111111","author_name":"X","rating":5}`
	resp, err := http.Post(srv.URL+"/api/reviews", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestSubmitReview_AlreadyReviewed(t *testing.T) {
	bookingID := "11111111-1111-1111-1111-111111111111"
	repo := &fakeRepo{saved: []Review{{ID: "existing", BookingID: bookingID}}}
	bookings := fakeBookingReader{bySlug: map[string]string{bookingID: "casadana"}}
	svc := newSvc(repo, bookings, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	body := `{"booking_id":"` + bookingID + `","author_name":"X","rating":5}`
	resp, err := http.Post(srv.URL+"/api/reviews", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}
}

func TestSubmitReview_BadRating(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	body := `{"booking_id":"11111111-1111-1111-1111-111111111111","author_name":"X","rating":99}`
	resp, err := http.Post(srv.URL+"/api/reviews", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestListReviewsByVilla(t *testing.T) {
	repo := &fakeRepo{saved: []Review{
		{ID: "1", VillaSlug: "casadana", Status: StatusPending},
		{ID: "2", VillaSlug: "casacasay", Status: StatusApproved},
	}}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/villas/casadana/reviews")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out listReviewsResponse
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if len(out.Reviews) != 1 {
		t.Errorf("len = %d, want 1", len(out.Reviews))
	}
}

func TestDeleteReview_NoContent(t *testing.T) {
	repo := &fakeRepo{saved: []Review{{ID: "abc"}}}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/reviews/abc", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
}

func TestDeleteReview_NotFound(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/reviews/missing", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}
