package pricing

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

func TestGetPricing_Empty(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/villas/casadana/pricing?from=2026-07-01&to=2026-08-01")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out pricingResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Overrides) != 0 {
		t.Errorf("overrides = %v, want []", out.Overrides)
	}
}

func TestGetPricing_WithOverrides(t *testing.T) {
	repo := &fakeRepo{overrides: []PriceOverride{
		{VillaSlug: "casadana", Date: d("2026-07-04"), PriceCents: 25000},
	}}
	svc := newSvc(repo, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/villas/casadana/pricing?from=2026-07-01&to=2026-08-01")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var out pricingResponse
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if len(out.Overrides) != 1 || out.Overrides[0].Date != "2026-07-04" || out.Overrides[0].PriceCents != 25000 {
		t.Errorf("unexpected overrides: %+v", out.Overrides)
	}
}

func TestGetPricing_UnknownVilla(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{}})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/villas/ghost/pricing?from=2026-07-01&to=2026-08-01")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestGetPricing_BadDates(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/villas/casadana/pricing?from=oops&to=2026-08-01")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestUpsertPricing_Created(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	body := `{"price_cents":25000,"dates":["2026-07-04","2026-07-05"]}`
	resp, err := http.Post(srv.URL+"/api/villas/casadana/pricing", "application/json",
		strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	var out upsertPricingResponse
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.Count != 2 {
		t.Errorf("count = %d, want 2", out.Count)
	}
}

func TestUpsertPricing_EmptyDates(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	body := `{"price_cents":100,"dates":[]}`
	resp, err := http.Post(srv.URL+"/api/villas/casadana/pricing", "application/json",
		strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestGetPricing_InvalidRange(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	// Valid date format but to <= from — should map to 422 INVALID_RANGE.
	resp, err := http.Get(srv.URL + "/api/villas/casadana/pricing?from=2026-08-01&to=2026-07-01")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}
