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
	httpserver.Register(ErrRuleNotFound, http.StatusNotFound, "SEASON_RULE_NOT_FOUND")
}

// Mount wires pricing routes. requireAuth guards every write (calendar
// overrides, base rate and fees, season rules); the three reads stay open so
// the public site can price a stay without a session.
func Mount(r chi.Router, svc *Service, requireAuth func(http.Handler) http.Handler) {
	r.Get("/api/villas/{slug}/pricing", listHandler(svc))
	r.Get("/api/villas/{slug}/pricing/settings", getSettingsHandler(svc))
	r.Get("/api/villas/{slug}/pricing/season-rules", listSeasonRulesHandler(svc))
	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Post("/api/villas/{slug}/pricing", upsertHandler(svc))
		r.Put("/api/villas/{slug}/pricing/settings", putSettingsHandler(svc))
		r.Post("/api/villas/{slug}/pricing/season-rules", createSeasonRuleHandler(svc))
		r.Patch("/api/pricing/season-rules/{id}", patchSeasonRuleHandler(svc))
		r.Delete("/api/pricing/season-rules/{id}", deleteSeasonRuleHandler(svc))
	})
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

type settingsDTO struct {
	BasePriceCents    int `json:"base_price_cents"`
	MinNights         int `json:"min_nights"`
	CleaningFeeCents  int `json:"cleaning_fee_cents"`
	ConciergeFeeCents int `json:"concierge_fee_cents"`
}

// Same cents ceiling as the per-date override — see upsertPricingRequest.
type putSettingsRequest struct {
	BasePriceCents    int `json:"base_price_cents"    validate:"min=0,max=100000000"`
	MinNights         int `json:"min_nights"          validate:"required,min=1,max=365"`
	CleaningFeeCents  int `json:"cleaning_fee_cents"  validate:"min=0,max=100000000"`
	ConciergeFeeCents int `json:"concierge_fee_cents" validate:"min=0,max=100000000"`
}

func settingsToDTO(s Settings) settingsDTO {
	return settingsDTO{
		BasePriceCents:    s.BasePriceCents,
		MinNights:         s.MinNights,
		CleaningFeeCents:  s.CleaningFeeCents,
		ConciergeFeeCents: s.ConciergeFeeCents,
	}
}

func getSettingsHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		settings, err := svc.GetSettings(r.Context(), slug)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(settingsToDTO(settings))
	}
}

func putSettingsHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")

		var req putSettingsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "invalid json: " + err.Error()})
			return
		}
		if err := validator.Struct(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: err.Error()})
			return
		}

		saved, err := svc.SaveSettings(r.Context(), Settings{
			VillaSlug:         slug,
			BasePriceCents:    req.BasePriceCents,
			MinNights:         req.MinNights,
			CleaningFeeCents:  req.CleaningFeeCents,
			ConciergeFeeCents: req.ConciergeFeeCents,
		})
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(settingsToDTO(saved))
	}
}

type seasonRuleDTO struct {
	ID         string `json:"id"`
	VillaSlug  string `json:"villa_slug"`
	Label      string `json:"label"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	PriceCents int    `json:"price_cents"`
}

type listSeasonRulesResponse struct {
	Rules []seasonRuleDTO `json:"rules"`
}

type createSeasonRuleRequest struct {
	Label      string `json:"label"       validate:"required,min=1,max=120"`
	StartDate  string `json:"start_date"  validate:"required,datetime=2006-01-02"`
	EndDate    string `json:"end_date"    validate:"required,datetime=2006-01-02"`
	PriceCents int    `json:"price_cents" validate:"min=0,max=100000000"`
}

// Every field is optional: a nil pointer means "leave this column alone".
type patchSeasonRuleRequest struct {
	Label      *string `json:"label"       validate:"omitempty,min=1,max=120"`
	StartDate  *string `json:"start_date"  validate:"omitempty,datetime=2006-01-02"`
	EndDate    *string `json:"end_date"    validate:"omitempty,datetime=2006-01-02"`
	PriceCents *int    `json:"price_cents" validate:"omitempty,min=0,max=100000000"`
}

func toDTO(rule *SeasonRule) seasonRuleDTO {
	return seasonRuleDTO{
		ID:         rule.ID,
		VillaSlug:  rule.VillaSlug,
		Label:      rule.Label,
		StartDate:  rule.Start.Format("2006-01-02"),
		EndDate:    rule.End.Format("2006-01-02"),
		PriceCents: rule.PriceCents,
	}
}

func listSeasonRulesHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		rules, err := svc.ListSeasonRules(r.Context(), slug)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		resp := listSeasonRulesResponse{Rules: make([]seasonRuleDTO, 0, len(rules))}
		for i := range rules {
			resp.Rules = append(resp.Rules, toDTO(&rules[i]))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func createSeasonRuleHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")

		var req createSeasonRuleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "invalid json: " + err.Error()})
			return
		}
		if err := validator.Struct(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: err.Error()})
			return
		}
		start, err := time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "start_date must be YYYY-MM-DD"})
			return
		}
		end, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "end_date must be YYYY-MM-DD"})
			return
		}

		rule, err := svc.CreateSeasonRule(r.Context(), CreateSeasonRuleCommand{
			VillaSlug:  slug,
			Label:      req.Label,
			Start:      start,
			End:        end,
			PriceCents: req.PriceCents,
		})
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(toDTO(rule))
	}
}

func patchSeasonRuleHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var req patchSeasonRuleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "invalid json: " + err.Error()})
			return
		}
		if err := validator.Struct(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: err.Error()})
			return
		}

		cmd := PatchSeasonRuleCommand{Label: req.Label, PriceCents: req.PriceCents}
		if req.StartDate != nil {
			start, err := time.Parse("2006-01-02", *req.StartDate)
			if err != nil {
				httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "start_date must be YYYY-MM-DD"})
				return
			}
			cmd.Start = &start
		}
		if req.EndDate != nil {
			end, err := time.Parse("2006-01-02", *req.EndDate)
			if err != nil {
				httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "end_date must be YYYY-MM-DD"})
				return
			}
			cmd.End = &end
		}

		rule, err := svc.UpdateSeasonRule(r.Context(), id, cmd)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(toDTO(rule))
	}
}

func deleteSeasonRuleHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := svc.DeleteSeasonRule(r.Context(), id); err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
