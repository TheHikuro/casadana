package review

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// openAuth stands in for a satisfied admin session so the route table itself
// is under test rather than the auth middleware.
func openAuth(next http.Handler) http.Handler { return next }

// closedAuth stands in for a missing admin session.
func closedAuth(http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
}

func newRouter(svc *Service) http.Handler {
	return newRouterWithAuth(svc, openAuth)
}

func newRouterWithAuth(svc *Service, requireAuth func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	Mount(r, svc, requireAuth)
	return r
}

func do(t *testing.T, method, url, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
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
	if out.BookingID == nil || *out.BookingID != bookingID {
		t.Errorf("booking_id = %v, want %q", out.BookingID, bookingID)
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

// The public listing must never leak a review an admin has not approved.
func TestListReviewsByVilla_ApprovedOnly(t *testing.T) {
	repo := &fakeRepo{saved: []Review{
		{ID: "1", VillaSlug: "casadana", Status: StatusApproved},
		{ID: "2", VillaSlug: "casadana", Status: StatusPending},
		{ID: "3", VillaSlug: "casadana", Status: StatusRejected},
		{ID: "4", VillaSlug: "casacasay", Status: StatusApproved},
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
		t.Fatalf("len = %d, want 1 (approved only)", len(out.Reviews))
	}
	if out.Reviews[0].Status != "approved" {
		t.Errorf("status = %q, want approved", out.Reviews[0].Status)
	}
}

func TestListReviewsForAdmin_AllStatuses(t *testing.T) {
	repo := &fakeRepo{saved: []Review{
		{ID: "1", VillaSlug: "casadana", Status: StatusApproved},
		{ID: "2", VillaSlug: "casadana", Status: StatusPending},
		{ID: "3", VillaSlug: "casadana", Status: StatusRejected},
	}}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	resp := do(t, http.MethodGet, srv.URL+"/api/admin/reviews?villa_slug=casadana", "")
	defer resp.Body.Close()
	var out listReviewsResponse
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if len(out.Reviews) != 3 {
		t.Fatalf("len = %d, want 3", len(out.Reviews))
	}
}

func TestListReviewsForAdmin_StatusFilter(t *testing.T) {
	repo := &fakeRepo{saved: []Review{
		{ID: "1", VillaSlug: "casadana", Status: StatusApproved},
		{ID: "2", VillaSlug: "casadana", Status: StatusPending},
	}}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	resp := do(t, http.MethodGet, srv.URL+"/api/admin/reviews?villa_slug=casadana&status=pending", "")
	defer resp.Body.Close()
	var out listReviewsResponse
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if len(out.Reviews) != 1 || out.Reviews[0].ID != "2" {
		t.Fatalf("reviews = %+v, want only the pending one", out.Reviews)
	}
}

func TestListReviewsForAdmin_BadStatus(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	resp := do(t, http.MethodGet, srv.URL+"/api/admin/reviews?villa_slug=casadana&status=archived", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestCreateAdminReview_Created(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	body := `{"villa_slug":"casadana","author_name":"Marta Ruiz","rating":5,"body":"Superbe","source":"airbnb","featured":true}`
	resp := do(t, http.MethodPost, srv.URL+"/api/admin/reviews", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	var out reviewDTO
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.BookingID != nil {
		t.Errorf("booking_id = %v, want null", *out.BookingID)
	}
	if out.Status != "approved" || !out.Featured || out.Source != "airbnb" {
		t.Errorf("unexpected response: %+v", out)
	}
}

func TestCreateAdminReview_BadPayload(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	resp := do(t, http.MethodPost, srv.URL+"/api/admin/reviews", `{"villa_slug":"casadana","rating":9}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestPatchReview_Moderates(t *testing.T) {
	repo := &fakeRepo{saved: []Review{
		{ID: "r1", VillaSlug: "casadana", AuthorName: "Marta Ruiz", Status: StatusPending},
	}}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	resp := do(t, http.MethodPatch, srv.URL+"/api/reviews/r1", `{"status":"approved","featured":true}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out reviewDTO
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Status != "approved" || !out.Featured {
		t.Errorf("unexpected response: %+v", out)
	}
}

func TestPatchReview_NotFound(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	resp := do(t, http.MethodPatch, srv.URL+"/api/reviews/ghost", `{"status":"approved"}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestPatchReview_BadStatus(t *testing.T) {
	repo := &fakeRepo{saved: []Review{{ID: "r1", VillaSlug: "casadana"}}}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	resp := do(t, http.MethodPatch, srv.URL+"/api/reviews/r1", `{"status":"archived"}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestDeleteReview_NoContent(t *testing.T) {
	repo := &fakeRepo{saved: []Review{{ID: "abc"}}}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	resp := do(t, http.MethodDelete, srv.URL+"/api/reviews/abc", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
}

func TestDeleteReview_NotFound(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	resp := do(t, http.MethodDelete, srv.URL+"/api/reviews/missing", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func getMeta(t *testing.T, url string) reviewMetaDTO {
	t.Helper()
	resp, err := http.Get(url + "/api/villas/casadana/reviews/meta")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out reviewMetaDTO
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestGetReviewMeta_NoApprovedReviewsIsZero(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	out := getMeta(t, srv.URL)
	if out.DisplayAvg != 0 || out.DisplayCount != 0 || out.Breakdown != (breakdownDTO{}) {
		t.Errorf("meta = %+v, want zero values", out)
	}
}

// The published figures are an average of the approved reviews and nothing
// else: a pending or hidden review must not move them.
func TestGetReviewMeta_CountsApprovedOnly(t *testing.T) {
	repo := &fakeRepo{saved: []Review{
		{ID: "r1", VillaSlug: "casadana", Status: StatusApproved, Rating: 5,
			Categories: CategoryRatings{Cleanliness: ptr(5.0), Host: ptr(4.0)}},
		{ID: "r2", VillaSlug: "casadana", Status: StatusApproved, Rating: 4,
			Categories: CategoryRatings{Cleanliness: ptr(4.0)}},
		{ID: "r3", VillaSlug: "casadana", Status: StatusPending, Rating: 1,
			Categories: CategoryRatings{Cleanliness: ptr(1.0)}},
		{ID: "r4", VillaSlug: "casadana", Status: StatusRejected, Rating: 1},
		{ID: "r5", VillaSlug: "casacasay", Status: StatusApproved, Rating: 1},
	}}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	out := getMeta(t, srv.URL)
	if out.DisplayCount != 2 {
		t.Errorf("display_count = %d, want 2 (approved only)", out.DisplayCount)
	}
	if out.DisplayAvg != 4.5 {
		t.Errorf("display_avg = %v, want 4.5", out.DisplayAvg)
	}
	if out.Breakdown.Cleanliness == nil || *out.Breakdown.Cleanliness != 4.5 {
		t.Errorf("cleanliness = %v, want 4.5", out.Breakdown.Cleanliness)
	}
	// Only r1 scored the host, so that one review is the whole average.
	if out.Breakdown.Host == nil || *out.Breakdown.Host != 4 {
		t.Errorf("host = %v, want 4", out.Breakdown.Host)
	}
	// Nobody scored comfort: an absent bar, not a zero-width one.
	if out.Breakdown.Comfort != nil {
		t.Errorf("comfort = %v, want null", *out.Breakdown.Comfort)
	}
}

// Moderating a review is what moves the published rating — the whole reason the
// figures are computed rather than typed in.
func TestGetReviewMeta_FollowsModeration(t *testing.T) {
	repo := &fakeRepo{saved: []Review{
		{ID: "r1", VillaSlug: "casadana", Status: StatusApproved, Rating: 5},
		{ID: "r2", VillaSlug: "casadana", Status: StatusPending, Rating: 3},
	}}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	if out := getMeta(t, srv.URL); out.DisplayAvg != 5 || out.DisplayCount != 1 {
		t.Fatalf("before = %+v, want avg 5 over 1 review", out)
	}

	resp := do(t, http.MethodPatch, srv.URL+"/api/reviews/r2", `{"status":"approved"}`)
	resp.Body.Close()

	if out := getMeta(t, srv.URL); out.DisplayAvg != 4 || out.DisplayCount != 2 {
		t.Errorf("after approving = %+v, want avg 4 over 2 reviews", out)
	}

	resp = do(t, http.MethodPatch, srv.URL+"/api/reviews/r1", `{"status":"rejected"}`)
	resp.Body.Close()

	if out := getMeta(t, srv.URL); out.DisplayAvg != 3 || out.DisplayCount != 1 {
		t.Errorf("after hiding = %+v, want avg 3 over 1 review", out)
	}
}

// The figures have no setter any more; the route only reads.
func TestPutReviewMetaIsGone(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	body := `{"display_avg":4.8,"display_count":42}`
	resp := do(t, http.MethodPut, srv.URL+"/api/villas/casadana/reviews/meta", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}

// The moderation surface — including DELETE, which used to be open to anyone —
// must sit behind the admin middleware.
func TestAdminRoutesRequireAuth(t *testing.T) {
	repo := &fakeRepo{saved: []Review{{ID: "r1", VillaSlug: "casadana"}}}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouterWithAuth(svc, closedAuth))
	defer srv.Close()

	guarded := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodDelete, path: "/api/reviews/r1"},
		{method: http.MethodPatch, path: "/api/reviews/r1", body: `{"status":"approved"}`},
		{method: http.MethodGet, path: "/api/admin/reviews?villa_slug=casadana"},
		{method: http.MethodPost, path: "/api/admin/reviews", body: `{"villa_slug":"casadana","author_name":"X","rating":5}`},
	}
	for _, tc := range guarded {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			resp := do(t, tc.method, srv.URL+tc.path, tc.body)
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", resp.StatusCode)
			}
		})
	}
	if len(repo.saved) != 1 {
		t.Errorf("saved count = %d, want 1 (nothing mutated)", len(repo.saved))
	}
}

func TestPublicRoutesStayOpen(t *testing.T) {
	repo := &fakeRepo{saved: []Review{{ID: "r1", VillaSlug: "casadana", Status: StatusApproved}}}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouterWithAuth(svc, closedAuth))
	defer srv.Close()

	for _, path := range []string{"/api/villas/casadana/reviews", "/api/villas/casadana/reviews/meta"} {
		t.Run(path, func(t *testing.T) {
			resp := do(t, http.MethodGet, srv.URL+path, "")
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("status = %d, want 200", resp.StatusCode)
			}
		})
	}
}
