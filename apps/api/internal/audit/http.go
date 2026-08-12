package audit

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/TheHikuro/casadana/internal/platform/httpserver"
)

func init() {
	httpserver.Register(ErrInvalidPayload, http.StatusUnprocessableEntity, "INVALID_PAYLOAD")
}

// Mount wires the history route. requireAuth guards it: the activity log is
// admin-only, there is no public counterpart.
func Mount(r chi.Router, svc *Service, requireAuth func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Get("/api/admin/history", listHistoryHandler(svc))
	})
}

type eventDTO struct {
	ID         string `json:"id"`
	VillaSlug  string `json:"villa_slug"`
	Type       string `json:"type"`
	Message    string `json:"message"`
	ActorEmail string `json:"actor_email"`
	CreatedAt  string `json:"created_at"`
}

type listHistoryResponse struct {
	Events []eventDTO `json:"events"`
	Total  int        `json:"total"`
	Page   int        `json:"page"`
	Limit  int        `json:"limit"`
}

func toDTO(e *Event) eventDTO {
	return eventDTO{
		ID:         e.ID,
		VillaSlug:  e.VillaSlug,
		Type:       string(e.Type),
		Message:    e.Message,
		ActorEmail: e.ActorEmail,
		CreatedAt:  e.CreatedAt.Format(time.RFC3339),
	}
}

func listHistoryHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

		events, total, err := svc.List(r.Context(), ListQuery{
			VillaSlug: r.URL.Query().Get("villa_slug"),
			Page:      page,
			Limit:     limit,
		})
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		page, limit, _ = normalizePaging(page, limit)

		resp := listHistoryResponse{
			Events: make([]eventDTO, 0, len(events)),
			Total:  total,
			Page:   page,
			Limit:  limit,
		}
		for i := range events {
			resp.Events = append(resp.Events, toDTO(&events[i]))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
