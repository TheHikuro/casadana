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

func Mount(r chi.Router, svc *Service) {
	r.Post("/api/reviews", submitHandler(svc))
	r.Get("/api/villas/{slug}/reviews", listByVillaHandler(svc))
	r.Delete("/api/reviews/{id}", deleteHandler(svc))
}

type submitReviewRequest struct {
	BookingID  string `json:"booking_id"  validate:"required,uuid"`
	AuthorName string `json:"author_name" validate:"required,min=1,max=120"`
	Rating     int    `json:"rating"      validate:"required,min=1,max=5"`
	Body       string `json:"body"        validate:"max=2000"`
}

type reviewDTO struct {
	ID         string `json:"id"`
	BookingID  string `json:"booking_id"`
	VillaSlug  string `json:"villa_slug"`
	AuthorName string `json:"author_name"`
	Rating     int    `json:"rating"`
	Body       string `json:"body"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
}

type listReviewsResponse struct {
	Reviews []reviewDTO `json:"reviews"`
}

func toDTO(r *Review) reviewDTO {
	return reviewDTO{
		ID:         r.ID,
		BookingID:  r.BookingID,
		VillaSlug:  r.VillaSlug,
		AuthorName: r.AuthorName,
		Rating:     r.Rating,
		Body:       r.Body,
		Status:     string(r.Status),
		CreatedAt:  r.CreatedAt.Format(time.RFC3339),
	}
}

func submitHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req submitReviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "invalid json: " + err.Error()})
			return
		}
		if err := validator.Struct(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: err.Error()})
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

func listByVillaHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		reviews, err := svc.ListByVilla(r.Context(), slug)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		resp := listReviewsResponse{Reviews: make([]reviewDTO, 0, len(reviews))}
		for i := range reviews {
			resp.Reviews = append(resp.Reviews, toDTO(&reviews[i]))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
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
