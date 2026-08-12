package audit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func newRouter(svc *Service, requireAuth func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	Mount(r, svc, requireAuth)
	return r
}

// noopAuth lets tests that aren't about auth exercise the admin-only route as
// if a valid session were already present.
func noopAuth(next http.Handler) http.Handler { return next }

// denyAllAuth simulates a middleware that rejects every request — used to
// prove requireAuth is actually wired onto the history route.
func denyAllAuth(http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
}

func seededSvc() *Service {
	repo := &fakeRepo{saved: []Event{
		{ID: "1", VillaSlug: "casadana", Type: TypePricing, Message: "Prix mis à jour",
			ActorEmail: "admin@casadana.fr", CreatedAt: d("2026-08-01")},
		{ID: "2", VillaSlug: "casadana", Type: TypeReview, Message: "Avis supprimé",
			ActorEmail: "admin@casadana.fr", CreatedAt: d("2026-07-31")},
		{ID: "3", VillaSlug: "casacasay", Type: TypeOwner, Message: "Propriétaire mis à jour",
			CreatedAt: d("2026-07-30")},
	}}
	return newSvc(repo)
}

func TestGetHistory_RequiresAuth(t *testing.T) {
	svc := seededSvc()
	srv := httptest.NewServer(newRouter(svc, denyAllAuth))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/admin/history?villa_slug=casadana")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestGetHistory_OK(t *testing.T) {
	svc := seededSvc()
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/admin/history?villa_slug=casadana")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out listHistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Events) != 2 || out.Total != 2 {
		t.Fatalf("events = %d, total = %d, want 2 and 2", len(out.Events), out.Total)
	}
	if out.Page != 1 || out.Limit != 20 {
		t.Errorf("page/limit = %d/%d, want 1/20", out.Page, out.Limit)
	}
	first := out.Events[0]
	if first.Type != "pricing" || first.ActorEmail != "admin@casadana.fr" {
		t.Errorf("unexpected event: %+v", first)
	}
	if _, err := time.Parse(time.RFC3339, first.CreatedAt); err != nil {
		t.Errorf("created_at = %q, want RFC3339: %v", first.CreatedAt, err)
	}
}

func TestGetHistory_EchoesNormalizedPaging(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantPage  int
		wantLimit int
		wantLen   int
	}{
		{"defaults", "", 1, 20, 2},
		{"page below 1", "&page=0&limit=5", 1, 5, 2},
		{"limit clamped", "&page=1&limit=999", 1, 100, 2},
		{"page beyond range", "&page=9&limit=1", 9, 1, 0},
		{"second page", "&page=2&limit=1", 2, 1, 1},
		{"garbage params fall back to defaults", "&page=abc&limit=xyz", 1, 20, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := seededSvc()
			srv := httptest.NewServer(newRouter(svc, noopAuth))
			defer srv.Close()

			resp, err := http.Get(srv.URL + "/api/admin/history?villa_slug=casadana" + tt.query)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			var out listHistoryResponse
			if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
				t.Fatal(err)
			}
			if out.Page != tt.wantPage || out.Limit != tt.wantLimit {
				t.Errorf("page/limit = %d/%d, want %d/%d", out.Page, out.Limit, tt.wantPage, tt.wantLimit)
			}
			if len(out.Events) != tt.wantLen {
				t.Errorf("events = %d, want %d", len(out.Events), tt.wantLen)
			}
			if out.Total != 2 {
				t.Errorf("total = %d, want 2 (unpaged count)", out.Total)
			}
		})
	}
}

func TestGetHistory_UnknownVilla(t *testing.T) {
	svc := seededSvc()
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	for _, q := range []string{"", "?villa_slug=", "?villa_slug=ghost"} {
		resp, err := http.Get(srv.URL + "/api/admin/history" + q)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("query %q: status = %d, want 422", q, resp.StatusCode)
		}
	}
}

func TestGetHistory_EmptyLogSerialisesAsArray(t *testing.T) {
	srv := httptest.NewServer(newRouter(newSvc(&fakeRepo{}), noopAuth))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/admin/history?villa_slug=casadana")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatal(err)
	}
	if string(raw["events"]) != "[]" {
		t.Errorf("events = %s, want []", raw["events"])
	}
}
