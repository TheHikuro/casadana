package review

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/TheHikuro/casadana/internal/platform/httpserver"
	"github.com/TheHikuro/casadana/internal/platform/validator"
)

func init() {
	httpserver.Register(ErrBookingNotFound, http.StatusNotFound, "BOOKING_NOT_FOUND")
	httpserver.Register(ErrAlreadyReviewed, http.StatusConflict, "ALREADY_REVIEWED")
	httpserver.Register(ErrNotFound, http.StatusNotFound, "REVIEW_NOT_FOUND")
	httpserver.Register(ErrInvalidPayload, http.StatusUnprocessableEntity, "INVALID_PAYLOAD")
}

// Mount wires review routes. requireAuth guards the moderation routes; guest
// submission and the two public reads stay open. The public villa listing
// returns approved reviews only — the admin listing is the one that sees
// pending and rejected rows. The meta route is read-only on purpose: the
// figures it serves are computed from those approved reviews, so there is
// nothing for an admin to write.
func Mount(r chi.Router, svc *Service, requireAuth func(http.Handler) http.Handler) {
	r.Post("/api/reviews", submitHandler(svc))
	r.Get("/api/villas/{slug}/reviews", listByVillaHandler(svc))
	r.Get("/api/villas/{slug}/reviews/meta", metaHandler(svc))
	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Delete("/api/reviews/{id}", deleteHandler(svc))
		r.Patch("/api/reviews/{id}", patchHandler(svc))
		r.Get("/api/admin/reviews", listForAdminHandler(svc))
		r.Post("/api/admin/reviews", createByAdminHandler(svc))
	})
}

type submitReviewRequest struct {
	BookingID  string `json:"booking_id"  validate:"required,uuid"`
	AuthorName string `json:"author_name" validate:"required,min=1,max=120"`
	Rating     int    `json:"rating"      validate:"required,min=1,max=5"`
	Body       string `json:"body"        validate:"max=2000"`
}

// categoryRatingsDTO carries the optional per-category scores. Every field is a
// pointer so that "not scored" stays distinguishable from a score of zero, both
// on the way in and on the way out.
type categoryRatingsDTO struct {
	Cleanliness *float64 `json:"cleanliness" validate:"omitempty,min=1,max=5"`
	Comfort     *float64 `json:"comfort"     validate:"omitempty,min=1,max=5"`
	Location    *float64 `json:"location"    validate:"omitempty,min=1,max=5"`
	Host        *float64 `json:"host"        validate:"omitempty,min=1,max=5"`
	Value       *float64 `json:"value"       validate:"omitempty,min=1,max=5"`
}

func (d categoryRatingsDTO) toDomain() CategoryRatings {
	return CategoryRatings{
		Cleanliness: d.Cleanliness,
		Comfort:     d.Comfort,
		Location:    d.Location,
		Host:        d.Host,
		Value:       d.Value,
	}
}

type createAdminReviewRequest struct {
	VillaSlug  string             `json:"villa_slug"  validate:"required,min=1,max=64"`
	AuthorName string             `json:"author_name" validate:"required,min=1,max=120"`
	Rating     int                `json:"rating"      validate:"required,min=1,max=5"`
	Body       string             `json:"body"        validate:"max=2000"`
	Status     string             `json:"status"      validate:"omitempty,oneof=pending approved rejected"`
	Meta       string             `json:"meta"        validate:"max=2000"`
	Source     string             `json:"source"      validate:"max=64"`
	Featured   bool               `json:"featured"`
	Categories categoryRatingsDTO `json:"categories"`
}

// patchReviewRequest is all-optional: a nil field means "leave unchanged".
type patchReviewRequest struct {
	Status     *string            `json:"status"   validate:"omitempty,oneof=pending approved rejected"`
	Featured   *bool              `json:"featured"`
	Meta       *string            `json:"meta"     validate:"omitempty,max=2000"`
	Source     *string            `json:"source"   validate:"omitempty,max=64"`
	Body       *string            `json:"body"     validate:"omitempty,max=2000"`
	Rating     *int               `json:"rating"   validate:"omitempty,min=1,max=5"`
	Categories categoryRatingsDTO `json:"categories"`
}

// breakdownDTO is the computed per-category average. A null means no approved
// review has scored that category, which the clients render as an absent bar
// rather than as a zero.
type breakdownDTO struct {
	Cleanliness *float64 `json:"cleanliness"`
	Comfort     *float64 `json:"comfort"`
	Location    *float64 `json:"location"`
	Host        *float64 `json:"host"`
	Value       *float64 `json:"value"`
}

type reviewMetaDTO struct {
	DisplayAvg   float64      `json:"display_avg"`
	DisplayCount int          `json:"display_count"`
	Breakdown    breakdownDTO `json:"breakdown"`
}

type reviewDTO struct {
	ID         string             `json:"id"`
	BookingID  *string            `json:"booking_id"`
	VillaSlug  string             `json:"villa_slug"`
	AuthorName string             `json:"author_name"`
	Rating     int                `json:"rating"`
	Body       string             `json:"body"`
	Status     string             `json:"status"`
	Meta       string             `json:"meta"`
	Source     string             `json:"source"`
	Featured   bool               `json:"featured"`
	Categories categoryRatingsDTO `json:"categories"`
	CreatedAt  string             `json:"created_at"`
}

type listReviewsResponse struct {
	Reviews []reviewDTO `json:"reviews"`
}

func toDTO(r *Review) reviewDTO {
	// booking_id is null for admin-authored reviews, not "".
	var bookingID *string
	if r.BookingID != "" {
		id := r.BookingID
		bookingID = &id
	}
	return reviewDTO{
		ID:         r.ID,
		BookingID:  bookingID,
		VillaSlug:  r.VillaSlug,
		AuthorName: r.AuthorName,
		Rating:     r.Rating,
		Body:       r.Body,
		Status:     string(r.Status),
		Meta:       r.Meta,
		Source:   r.Source,
		Featured: r.Featured,
		Categories: categoryRatingsDTO{
			Cleanliness: r.Categories.Cleanliness,
			Comfort:     r.Categories.Comfort,
			Location:    r.Categories.Location,
			Host:        r.Categories.Host,
			Value:       r.Categories.Value,
		},
		CreatedAt: r.CreatedAt.Format(time.RFC3339),
	}
}

func toMetaDTO(m ReviewMeta) reviewMetaDTO {
	return reviewMetaDTO{
		DisplayAvg:   m.DisplayAvg,
		DisplayCount: m.DisplayCount,
		Breakdown: breakdownDTO{
			Cleanliness: m.Breakdown.Cleanliness,
			Comfort:     m.Breakdown.Comfort,
			Location:    m.Breakdown.Location,
			Host:        m.Breakdown.Host,
			Value:       m.Breakdown.Value,
		},
	}
}

func writeReviews(w http.ResponseWriter, reviews []Review) {
	resp := listReviewsResponse{Reviews: make([]reviewDTO, 0, len(reviews))}
	for i := range reviews {
		resp.Reviews = append(resp.Reviews, toDTO(&reviews[i]))
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func decodeAndValidate(w http.ResponseWriter, r *http.Request, req any) bool {
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "invalid json: " + err.Error()})
		return false
	}
	if err := validator.Struct(req); err != nil {
		httpserver.WriteError(w, r, &httpserver.ValidationError{Message: err.Error()})
		return false
	}
	return true
}

func submitHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req submitReviewRequest
		if !decodeAndValidate(w, r, &req) {
			return
		}
		rv, err := svc.Submit(r.Context(), SubmitCommand{
			BookingID:  req.BookingID,
			AuthorName: req.AuthorName,
			Rating:     req.Rating,
			Body:       req.Body,
		})
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(toDTO(rv))
	}
}

func createByAdminHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createAdminReviewRequest
		if !decodeAndValidate(w, r, &req) {
			return
		}
		rv, err := svc.CreateByAdmin(r.Context(), CreateByAdminCommand{
			VillaSlug:  req.VillaSlug,
			AuthorName: req.AuthorName,
			Rating:     req.Rating,
			Body:       req.Body,
			Status:     Status(req.Status),
			Meta:       req.Meta,
			Source:     req.Source,
			Featured:   req.Featured,
			Categories: req.Categories.toDomain(),
		})
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(toDTO(rv))
	}
}

func listByVillaHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		reviews, err := svc.ListByVilla(r.Context(), slug)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		writeReviews(w, reviews)
	}
}

func listForAdminHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var statusFilter *Status
		if s := r.URL.Query().Get("status"); s != "" {
			switch Status(s) {
			case StatusPending, StatusApproved, StatusRejected:
				st := Status(s)
				statusFilter = &st
			default:
				httpserver.WriteError(w, r, &httpserver.ValidationError{
					Message: "status must be one of: pending, approved, rejected",
				})
				return
			}
		}
		reviews, err := svc.ListForAdmin(r.Context(), r.URL.Query().Get("villa_slug"), statusFilter)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		writeReviews(w, reviews)
	}
}

func patchHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var req patchReviewRequest
		if !decodeAndValidate(w, r, &req) {
			return
		}
		patch := UpdatePatch{
			Featured:   req.Featured,
			Meta:       req.Meta,
			Source:     req.Source,
			Body:       req.Body,
			Rating:     req.Rating,
			Categories: req.Categories.toDomain(),
		}
		if req.Status != nil {
			st := Status(*req.Status)
			patch.Status = &st
		}
		rv, err := svc.Update(r.Context(), id, patch)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(toDTO(rv))
	}
}

func deleteHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := svc.Delete(r.Context(), id); err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func metaHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		meta, err := svc.Meta(r.Context(), slug)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(toMetaDTO(meta))
	}
}

