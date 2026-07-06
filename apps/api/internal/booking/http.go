package booking

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/TheHikuro/casadana/internal/platform/httpserver"
	"github.com/TheHikuro/casadana/internal/platform/validator"
)

func init() {
	httpserver.Register(ErrDatesConflict, http.StatusConflict, "DATES_CONFLICT")
	httpserver.Register(ErrUnknownVilla, http.StatusNotFound, "UNKNOWN_VILLA")
	httpserver.Register(ErrInvalidStatus, http.StatusConflict, "INVALID_STATUS")
	httpserver.Register(ErrNotFound, http.StatusNotFound, "BOOKING_NOT_FOUND")
}

// Mount wires booking routes. requireAuth guards the admin-only routes
// (list/patch/delete); POST (public booking creation) and the availability
// read stay open for unauthenticated guests.
func Mount(r chi.Router, svc *Service, requireAuth func(http.Handler) http.Handler) {
	r.Post("/api/bookings", createHandler(svc))
	r.Get("/api/villas/{slug}/availability", availabilityHandler(svc))
	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Get("/api/bookings", listBookingsHandler(svc))
		r.Patch("/api/bookings/{id}", patchBookingHandler(svc))
		r.Delete("/api/bookings/{id}", deleteBookingHandler(svc))
	})
}

type patchBookingRequest struct {
	Status string `json:"status" validate:"required,oneof=approved rejected cancelled paid"`
}

type createBookingRequest struct {
	VillaSlug  string `json:"villa_slug"  validate:"required,min=1,max=64"`
	GuestName  string `json:"guest_name"  validate:"required,min=1,max=120"`
	GuestEmail string `json:"guest_email" validate:"required,email,max=255"`
	GuestPhone string `json:"guest_phone" validate:"required,min=2,max=40"`
	CheckIn    string `json:"check_in"    validate:"required,datetime=2006-01-02"`
	CheckOut   string `json:"check_out"   validate:"required,datetime=2006-01-02"`
	Adults     int    `json:"adults"      validate:"required,min=1,max=20"`
	Children   int    `json:"children"    validate:"min=0,max=20"`
	Message    string `json:"message"     validate:"max=2000"`
	Source     string `json:"source"      validate:"omitempty,oneof=direct airbnb booking_com"`
}

type bookingResponse struct {
	ID         string `json:"id"`
	VillaSlug  string `json:"villa_slug"`
	Status     string `json:"status"`
	CheckIn    string `json:"check_in"`
	CheckOut   string `json:"check_out"`
	GuestName  string `json:"guest_name"`
	GuestEmail string `json:"guest_email"`
	Source     string `json:"source"`
	Adults     int    `json:"adults"`
	Children   int    `json:"children"`
	CreatedAt  string `json:"created_at"`
}

func toResponse(b *Booking) bookingResponse {
	return bookingResponse{
		ID:         b.ID,
		VillaSlug:  b.VillaSlug,
		Status:     string(b.Status),
		CheckIn:    b.CheckIn.Format("2006-01-02"),
		CheckOut:   b.CheckOut.Format("2006-01-02"),
		GuestName:  b.GuestName,
		GuestEmail: b.GuestEmail,
		Source:     b.Source,
		Adults:     b.Adults,
		Children:   b.Children,
		CreatedAt:  b.CreatedAt.Format(time.RFC3339),
	}
}

func createHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createBookingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "invalid json: " + err.Error()})
			return
		}
		if err := validator.Struct(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: err.Error()})
			return
		}
		checkIn, err := time.Parse("2006-01-02", req.CheckIn)
		if err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "check_in must be YYYY-MM-DD"})
			return
		}
		checkOut, err := time.Parse("2006-01-02", req.CheckOut)
		if err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "check_out must be YYYY-MM-DD"})
			return
		}

		b, err := svc.Create(r.Context(), CreateCommand{
			VillaSlug:  req.VillaSlug,
			GuestName:  req.GuestName,
			GuestEmail: req.GuestEmail,
			GuestPhone: req.GuestPhone,
			CheckIn:    checkIn,
			CheckOut:   checkOut,
			Adults:     req.Adults,
			Children:   req.Children,
			Message:    req.Message,
			Source:     req.Source,
		})
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(toResponse(b))
	}
}

func patchBookingHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var req patchBookingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "invalid json: " + err.Error()})
			return
		}
		if err := validator.Struct(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: err.Error()})
			return
		}
		b, err := svc.TransitionStatus(r.Context(), id, Status(req.Status))
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(toResponse(b))
	}
}

type bookedRangeDTO struct {
	CheckIn  string `json:"check_in"`
	CheckOut string `json:"check_out"`
}

type availabilityResponse struct {
	BookedRanges  []bookedRangeDTO `json:"booked_ranges"`
	PendingRanges []bookedRangeDTO `json:"pending_ranges"`
}

func toRangeDTOs(ranges []DateRange) []bookedRangeDTO {
	out := make([]bookedRangeDTO, 0, len(ranges))
	for _, rng := range ranges {
		out = append(out, bookedRangeDTO{
			CheckIn:  rng.CheckIn.Format("2006-01-02"),
			CheckOut: rng.CheckOut.Format("2006-01-02"),
		})
	}
	return out
}

func availabilityHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		from, err1 := time.Parse("2006-01-02", r.URL.Query().Get("from"))
		to, err2 := time.Parse("2006-01-02", r.URL.Query().Get("to"))
		if err1 != nil || err2 != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "from and to must be YYYY-MM-DD"})
			return
		}
		avail, err := svc.Availability(r.Context(), slug, from, to)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		resp := availabilityResponse{
			BookedRanges:  toRangeDTOs(avail.Booked),
			PendingRanges: toRangeDTOs(avail.Pending),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

type listBookingsResponse struct {
	Bookings []bookingResponse `json:"bookings"`
	Page     int               `json:"page"`
	Limit    int               `json:"limit"`
	Total    int               `json:"total"`
}

func listBookingsHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

		var villaSlugFilter *string
		if vs := r.URL.Query().Get("villa_slug"); vs != "" {
			villaSlugFilter = &vs
		}

		var statusFilter *Status
		if s := r.URL.Query().Get("status"); s != "" {
			switch Status(s) {
			case StatusPending, StatusApproved, StatusRejected, StatusCancelled, StatusPaid:
				st := Status(s)
				statusFilter = &st
			default:
				httpserver.WriteError(w, r, &httpserver.ValidationError{
					Message: "status must be one of: pending, approved, rejected, cancelled, paid",
				})
				return
			}
		}

		bookings, total, err := svc.List(r.Context(), villaSlugFilter, statusFilter, page, limit)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		if page < 1 {
			page = 1
		}
		if limit < 1 {
			limit = 20
		}
		if limit > 100 {
			limit = 100
		}

		resp := listBookingsResponse{
			Bookings: make([]bookingResponse, 0, len(bookings)),
			Page:     page,
			Limit:    limit,
			Total:    total,
		}
		for i := range bookings {
			resp.Bookings = append(resp.Bookings, toResponse(&bookings[i]))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func deleteBookingHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := svc.Delete(r.Context(), id); err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
