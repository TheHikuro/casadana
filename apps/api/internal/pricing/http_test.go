package pricing

import (
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
// prove requireAuth is actually wired onto the writes (and not onto the
// public reads).
func denyAllAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
}

func TestGetPricing_Empty(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc, noopAuth))
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
	srv := httptest.NewServer(newRouter(svc, noopAuth))
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

// The public booking panel prices a stay off `nights`, so the serialized
// contract is asserted here and not only at the service boundary.
func TestGetPricing_ServesResolvedNights(t *testing.T) {
	repo := &fakeRepo{
		overrides: []PriceOverride{
			{VillaSlug: "casadana", Date: d("2026-07-02"), PriceCents: 25000},
		},
		rules: []SeasonRule{
			{ID: "r1", VillaSlug: "casadana", Label: "Summer", Start: d("2026-07-01"), End: d("2026-07-02"), PriceCents: 12000},
		},
		settings: map[string]Settings{
			"casadana": {VillaSlug: "casadana", BasePriceCents: 8500, MinNights: 2},
		},
	}
	svc := newSvc(repo, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/villas/casadana/pricing?from=2026-07-01&to=2026-07-04")
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
	want := []nightlyPriceDTO{
		{Date: "2026-07-01", PriceCents: 12000, Label: "Summer"},
		{Date: "2026-07-02", PriceCents: 25000, Label: OverrideLabel},
		{Date: "2026-07-03", PriceCents: 8500, Label: StandardLabel},
	}
	if len(out.Nights) != len(want) {
		t.Fatalf("nights = %+v, want %d entries", out.Nights, len(want))
	}
	for i, w := range want {
		if out.Nights[i] != w {
			t.Errorf("nights[%d] = %+v, want %+v", i, out.Nights[i], w)
		}
	}
}

func TestGetPricing_WindowTooLarge(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/villas/casadana/pricing?from=2026-01-01&to=2030-01-01")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestGetPricing_UnknownVilla(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{}})
	srv := httptest.NewServer(newRouter(svc, noopAuth))
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
	srv := httptest.NewServer(newRouter(svc, noopAuth))
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
	srv := httptest.NewServer(newRouter(svc, noopAuth))
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
	srv := httptest.NewServer(newRouter(svc, noopAuth))
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
	srv := httptest.NewServer(newRouter(svc, noopAuth))
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

// do issues a request the stdlib helpers don't cover (PUT, PATCH, DELETE).
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

func TestUpsertPricing_RequiresAdmin(t *testing.T) {
	// POST /pricing used to be unauthenticated — it must now sit behind the
	// admin middleware.
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc, denyAllAuth))
	defer srv.Close()

	body := `{"price_cents":25000,"dates":["2026-07-04"]}`
	resp, err := http.Post(srv.URL+"/api/villas/casadana/pricing", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestPublicReads_StayOpen(t *testing.T) {
	repo := &fakeRepo{rules: []SeasonRule{{ID: "r1", VillaSlug: "casadana", Label: "Summer peak",
		Start: d("2026-07-01"), End: d("2026-08-31"), PriceCents: 25000}}}
	svc := newSvc(repo, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc, denyAllAuth))
	defer srv.Close()

	for _, path := range []string{
		"/api/villas/casadana/pricing?from=2026-07-01&to=2026-08-01",
		"/api/villas/casadana/pricing/settings",
		"/api/villas/casadana/pricing/season-rules",
	} {
		t.Run(path, func(t *testing.T) {
			resp, err := http.Get(srv.URL + path)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", resp.StatusCode)
			}
		})
	}
}

func TestGetSettings_NeverConfiguredIsZeroed(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/villas/casadana/pricing/settings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out settingsDTO
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if (out != settingsDTO{}) {
		t.Errorf("settings = %+v, want all zeros", out)
	}
}

func TestGetSettings_UnknownVillaIs404(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{}})
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/villas/ghost/pricing/settings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestPutSettings(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"happy", `{"base_price_cents":18500,"min_nights":3,"cleaning_fee_cents":8000,"concierge_fee_cents":5000}`, http.StatusOK},
		{"missing min_nights", `{"base_price_cents":18500}`, http.StatusUnprocessableEntity},
		{"negative fee", `{"base_price_cents":18500,"min_nights":3,"cleaning_fee_cents":-1}`, http.StatusUnprocessableEntity},
		{"invalid json", `{`, http.StatusUnprocessableEntity},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
			srv := httptest.NewServer(newRouter(svc, noopAuth))
			defer srv.Close()

			resp := do(t, http.MethodPut, srv.URL+"/api/villas/casadana/pricing/settings", c.body)
			defer resp.Body.Close()
			if resp.StatusCode != c.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, c.wantStatus)
			}
			if c.wantStatus != http.StatusOK {
				return
			}
			var out settingsDTO
			if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
				t.Fatal(err)
			}
			want := settingsDTO{BasePriceCents: 18500, MinNights: 3, CleaningFeeCents: 8000, ConciergeFeeCents: 5000}
			if out != want {
				t.Errorf("settings = %+v, want %+v", out, want)
			}
		})
	}
}

func TestPutSettings_RequiresAdmin(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc, denyAllAuth))
	defer srv.Close()

	resp := do(t, http.MethodPut, srv.URL+"/api/villas/casadana/pricing/settings",
		`{"base_price_cents":18500,"min_nights":3}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestListSeasonRules(t *testing.T) {
	repo := &fakeRepo{rules: []SeasonRule{
		{ID: "r1", VillaSlug: "casadana", Label: "Summer peak", Start: d("2026-07-01"), End: d("2026-08-31"), PriceCents: 25000},
		{ID: "r2", VillaSlug: "casacasay", Label: "Other villa", Start: d("2026-07-01"), End: d("2026-08-31"), PriceCents: 18000},
	}}
	svc := newSvc(repo, fakeAllowlist{allowed: map[string]bool{"casadana": true, "casacasay": true}})
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/villas/casadana/pricing/season-rules")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var out listSeasonRulesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Rules) != 1 {
		t.Fatalf("rules = %d, want 1", len(out.Rules))
	}
	want := seasonRuleDTO{ID: "r1", VillaSlug: "casadana", Label: "Summer peak",
		StartDate: "2026-07-01", EndDate: "2026-08-31", PriceCents: 25000}
	if out.Rules[0] != want {
		t.Errorf("rule = %+v, want %+v", out.Rules[0], want)
	}
}

func TestCreateSeasonRule_Created(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	body := `{"label":"Summer peak","start_date":"2026-07-01","end_date":"2026-08-31","price_cents":25000}`
	resp, err := http.Post(srv.URL+"/api/villas/casadana/pricing/season-rules", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	var out seasonRuleDTO
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.ID == "" || out.VillaSlug != "casadana" || out.StartDate != "2026-07-01" || out.EndDate != "2026-08-31" {
		t.Errorf("unexpected rule: %+v", out)
	}
}

func TestCreateSeasonRule_Rejected(t *testing.T) {
	cases := []struct {
		name       string
		slug       string
		body       string
		wantStatus int
	}{
		{"unknown villa", "ghost", `{"label":"x","start_date":"2026-07-01","end_date":"2026-07-02","price_cents":100}`, http.StatusNotFound},
		{"empty label", "casadana", `{"label":"","start_date":"2026-07-01","end_date":"2026-07-02","price_cents":100}`, http.StatusUnprocessableEntity},
		{"bad date", "casadana", `{"label":"x","start_date":"oops","end_date":"2026-07-02","price_cents":100}`, http.StatusUnprocessableEntity},
		{"end before start", "casadana", `{"label":"x","start_date":"2026-07-02","end_date":"2026-07-01","price_cents":100}`, http.StatusUnprocessableEntity},
		{"negative price", "casadana", `{"label":"x","start_date":"2026-07-01","end_date":"2026-07-02","price_cents":-1}`, http.StatusUnprocessableEntity},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
			srv := httptest.NewServer(newRouter(svc, noopAuth))
			defer srv.Close()

			resp, err := http.Post(srv.URL+"/api/villas/"+c.slug+"/pricing/season-rules", "application/json",
				strings.NewReader(c.body))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != c.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, c.wantStatus)
			}
		})
	}
}

func TestCreateSeasonRule_RequiresAdmin(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc, denyAllAuth))
	defer srv.Close()

	body := `{"label":"Summer peak","start_date":"2026-07-01","end_date":"2026-08-31","price_cents":25000}`
	resp, err := http.Post(srv.URL+"/api/villas/casadana/pricing/season-rules", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestPatchSeasonRule(t *testing.T) {
	cases := []struct {
		name       string
		id         string
		body       string
		wantStatus int
		wantLabel  string
		wantPrice  int
		wantStart  string
	}{
		{"label only", "r1", `{"label":"Renamed"}`, http.StatusOK, "Renamed", 25000, "2026-07-01"},
		{"price only", "r1", `{"price_cents":30000}`, http.StatusOK, "Summer peak", 30000, "2026-07-01"},
		{"dates only", "r1", `{"start_date":"2026-07-10"}`, http.StatusOK, "Summer peak", 25000, "2026-07-10"},
		{"empty patch keeps everything", "r1", `{}`, http.StatusOK, "Summer peak", 25000, "2026-07-01"},
		{"unknown rule", "r9", `{"price_cents":1}`, http.StatusNotFound, "", 0, ""},
		{"inverted range", "r1", `{"start_date":"2026-09-30"}`, http.StatusUnprocessableEntity, "", 0, ""},
		{"blank label", "r1", `{"label":""}`, http.StatusUnprocessableEntity, "", 0, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo := &fakeRepo{rules: []SeasonRule{{ID: "r1", VillaSlug: "casadana", Label: "Summer peak",
				Start: d("2026-07-01"), End: d("2026-08-31"), PriceCents: 25000}}}
			svc := newSvc(repo, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
			srv := httptest.NewServer(newRouter(svc, noopAuth))
			defer srv.Close()

			resp := do(t, http.MethodPatch, srv.URL+"/api/pricing/season-rules/"+c.id, c.body)
			defer resp.Body.Close()
			if resp.StatusCode != c.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, c.wantStatus)
			}
			if c.wantStatus != http.StatusOK {
				return
			}
			var out seasonRuleDTO
			if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
				t.Fatal(err)
			}
			if out.Label != c.wantLabel || out.PriceCents != c.wantPrice || out.StartDate != c.wantStart {
				t.Errorf("rule = %+v, want label %q price %d start %s", out, c.wantLabel, c.wantPrice, c.wantStart)
			}
		})
	}
}

func TestPatchSeasonRule_RequiresAdmin(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc, denyAllAuth))
	defer srv.Close()

	resp := do(t, http.MethodPatch, srv.URL+"/api/pricing/season-rules/r1", `{"price_cents":1}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestDeleteSeasonRule_NoContent(t *testing.T) {
	repo := &fakeRepo{rules: []SeasonRule{{ID: "r1", VillaSlug: "casadana", Label: "Summer peak",
		Start: d("2026-07-01"), End: d("2026-08-31"), PriceCents: 25000}}}
	svc := newSvc(repo, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	resp := do(t, http.MethodDelete, srv.URL+"/api/pricing/season-rules/r1", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}

	resp2 := do(t, http.MethodDelete, srv.URL+"/api/pricing/season-rules/r1", "")
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp2.StatusCode)
	}
}

func TestDeleteSeasonRule_RequiresAdmin(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc, denyAllAuth))
	defer srv.Close()

	resp := do(t, http.MethodDelete, srv.URL+"/api/pricing/season-rules/r1", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}
