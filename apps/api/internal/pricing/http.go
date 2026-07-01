package pricing

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/TheHikuro/casadana/internal/platform/httpserver"
	"github.com/TheHikuro/casadana/internal/platform/validator"
)

func init() {
	httpserver.Register(ErrUnknownVilla, http.StatusNotFound, "UNKNOWN_VILLA")
	httpserver.Register(ErrInvalidRange, http.StatusUnprocessableEntity, "INVALID_RANGE")
	httpserver.Register(ErrInvalidPayload, http.StatusUnprocessableEntity, "INVALID_PAYLOAD")
}

func Mount(r chi.Router, svc *Service) {
	r.Get("/api/villas/{slug}/pricing", listHandler(svc))
	r.Post("/api/villas/{slug}/pricing", upsertHandler(svc))
}

type priceOverrideDTO struct {
	Date       string `json:"date"`
	PriceCents int    `json:"price_cents"`
}

type pricingResponse struct {
	Overrides []priceOverrideDTO `json:"overrides"`
}

func listHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		from, errFrom := time.Parse("2006-01-02", r.URL.Query().Get("from"))
		to, errTo := time.Parse("2006-01-02", r.URL.Query().Get("to"))
		if errFrom != nil || errTo != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{
				Message: "from and to must be YYYY-MM-DD",
			})
			return
		}

		overrides, err := svc.ListOverrides(r.Context(), slug, from, to)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}

		resp := pricingResponse{Overrides: make([]priceOverrideDTO, 0, len(overrides))}
		for _, o := range overrides {
			resp.Overrides = append(resp.Overrides, priceOverrideDTO{
				Date:       o.Date.Format("2006-01-02"),
				PriceCents: o.PriceCents,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

type upsertPricingRequest struct {
	// PriceCents max = 100_000_000 (€1,000,000/night). Prevents accidental
	// int32 overflow at the DB column (int32 max is ~2.1B cents) and rules
	// out nonsense values from typos.
	PriceCents int      `json:"price_cents" validate:"min=0,max=100000000"`
	Dates      []string `json:"dates"       validate:"required,min=1,dive,datetime=2006-01-02"`
}

type upsertPricingResponse struct {
	Count int `json:"count"`
}

func upsertHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")

		var req upsertPricingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "invalid json: " + err.Error()})
			return
		}
		if err := validator.Struct(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: err.Error()})
			return
		}

		dates := make([]time.Time, 0, len(req.Dates))
		for _, ds := range req.Dates {
			t, err := time.Parse("2006-01-02", ds)
			if err != nil {
				httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "invalid date: " + ds})
				return
			}
			dates = append(dates, t)
		}

		count, err := svc.UpsertOverrides(r.Context(), slug, req.PriceCents, dates)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(upsertPricingResponse{Count: count})
	}
}
