package booking

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TheHikuro/casadana/internal/db"
)

type pgRepo struct {
	pool *pgxpool.Pool
}

func NewPgRepo(pool *pgxpool.Pool) Repository { return &pgRepo{pool: pool} }

func (r *pgRepo) q() *db.Queries { return db.New(r.pool) }

func (r *pgRepo) Save(ctx context.Context, b *Booking) error {
	id, err := uuid.Parse(b.ID)
	if err != nil {
		return fmt.Errorf("booking: invalid id: %w", err)
	}
	_, err = r.q().InsertBooking(ctx, db.InsertBookingParams{
		ID:         pgtype.UUID{Bytes: [16]byte(id), Valid: true},
		VillaSlug:  b.VillaSlug,
		GuestName:  b.GuestName,
		GuestEmail: b.GuestEmail,
		GuestPhone: b.GuestPhone,
		CheckIn:    pgtype.Date{Time: b.CheckIn, Valid: true},
		CheckOut:   pgtype.Date{Time: b.CheckOut, Valid: true},
		Adults:     int16(b.Adults),
		Children:   int16(b.Children),
		Message:    b.Message,
		Source:     b.Source,
		Status:     db.BookingStatus(b.Status),
	})
	return err
}

// FindOverlapping returns existing bookings overlapping the [from, to) window.
//
// The sqlc-generated FindOverlappingBookingsParams has fields named after the
// SQL columns being compared, not after their semantic role. The SQL is:
//
//	check_in  < $3   (param order: $1=VillaSlug, $2=CheckOut, $3=CheckIn)
//	check_out > $2
//
// So the `CheckIn` field is bound to $3 (the upper bound: query `to`), and
// the `CheckOut` field is bound to $2 (the lower bound: query `from`).
func (r *pgRepo) FindOverlapping(ctx context.Context, villaSlug string, from, to time.Time) ([]Booking, error) {
	rows, err := r.q().FindOverlappingBookings(ctx, db.FindOverlappingBookingsParams{
		VillaSlug: villaSlug,
		CheckOut:  pgtype.Date{Time: from, Valid: true},
		CheckIn:   pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	out := make([]Booking, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToBooking(row))
	}
	return out, nil
}

// FindOverlappingConfirmed mirrors the same param naming quirk as
// FindOverlapping, plus excludeID bound to $4.
func (r *pgRepo) FindOverlappingConfirmed(ctx context.Context, villaSlug string, from, to time.Time, excludeID string) ([]Booking, error) {
	excludeUUID, err := uuid.Parse(excludeID)
	if err != nil {
		return nil, fmt.Errorf("booking: invalid exclude id: %w", err)
	}
	rows, err := r.q().FindOverlappingConfirmedBookings(ctx, db.FindOverlappingConfirmedBookingsParams{
		VillaSlug: villaSlug,
		CheckOut:  pgtype.Date{Time: from, Valid: true},
		CheckIn:   pgtype.Date{Time: to, Valid: true},
		ID:        pgtype.UUID{Bytes: [16]byte(excludeUUID), Valid: true},
	})
	if err != nil {
		return nil, err
	}
	out := make([]Booking, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToBooking(row))
	}
	return out, nil
}

// BookedRanges mirrors the same param naming quirk as FindOverlapping.
func (r *pgRepo) BookedRanges(ctx context.Context, villaSlug string, from, to time.Time) ([]DateRange, error) {
	rows, err := r.q().ListBookedRanges(ctx, db.ListBookedRangesParams{
		VillaSlug: villaSlug,
		CheckOut:  pgtype.Date{Time: from, Valid: true},
		CheckIn:   pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	out := make([]DateRange, 0, len(rows))
	for _, row := range rows {
		out = append(out, DateRange{CheckIn: row.CheckIn.Time, CheckOut: row.CheckOut.Time})
	}
	return out, nil
}

// PendingRanges mirrors the same param naming quirk as FindOverlapping.
func (r *pgRepo) PendingRanges(ctx context.Context, villaSlug string, from, to time.Time) ([]DateRange, error) {
	rows, err := r.q().ListPendingRanges(ctx, db.ListPendingRangesParams{
		VillaSlug: villaSlug,
		CheckOut:  pgtype.Date{Time: from, Valid: true},
		CheckIn:   pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	out := make([]DateRange, 0, len(rows))
	for _, row := range rows {
		out = append(out, DateRange{CheckIn: row.CheckIn.Time, CheckOut: row.CheckOut.Time})
	}
	return out, nil
}

func (r *pgRepo) Get(ctx context.Context, id string) (*Booking, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("booking: invalid id: %w", err)
	}
	row, err := r.q().GetBookingByID(ctx, pgtype.UUID{Bytes: [16]byte(uid), Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	b := rowToBooking(row)
	return &b, nil
}

func (r *pgRepo) UpdateStatus(ctx context.Context, id string, status Status, updatedAt time.Time) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("booking: invalid id: %w", err)
	}
	return r.q().UpdateBookingStatus(ctx, db.UpdateBookingStatusParams{
		ID:        pgtype.UUID{Bytes: [16]byte(uid), Valid: true},
		Status:    db.BookingStatus(status),
		UpdatedAt: pgtype.Timestamptz{Time: updatedAt, Valid: true},
	})
}

func rowToBooking(row db.Booking) Booking {
	idStr := ""
	if row.ID.Valid {
		u, _ := uuid.FromBytes(row.ID.Bytes[:])
		idStr = u.String()
	}
	return Booking{
		ID:         idStr,
		VillaSlug:  row.VillaSlug,
		GuestName:  row.GuestName,
		GuestEmail: row.GuestEmail,
		GuestPhone: row.GuestPhone,
		CheckIn:    row.CheckIn.Time,
		CheckOut:   row.CheckOut.Time,
		Adults:     int(row.Adults),
		Children:   int(row.Children),
		Message:    row.Message,
		Status:     Status(row.Status),
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
		Source:     row.Source,
	}
}

func (r *pgRepo) List(ctx context.Context, villaSlug *string, status *Status, limit, offset int) ([]Booking, error) {
	switch {
	case villaSlug != nil && status != nil:
		rows, err := r.q().ListBookingsPagedByVillaAndStatus(ctx, db.ListBookingsPagedByVillaAndStatusParams{
			VillaSlug: *villaSlug,
			Status:    db.BookingStatus(*status),
			Limit:     int32(limit),
			Offset:    int32(offset),
		})
		return mapBookingRows(rows, err)
	case villaSlug != nil:
		rows, err := r.q().ListBookingsPagedByVilla(ctx, db.ListBookingsPagedByVillaParams{
			VillaSlug: *villaSlug,
			Limit:     int32(limit),
			Offset:    int32(offset),
		})
		return mapBookingRows(rows, err)
	case status != nil:
		rows, err := r.q().ListBookingsPagedByStatus(ctx, db.ListBookingsPagedByStatusParams{
			Status: db.BookingStatus(*status),
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		return mapBookingRows(rows, err)
	default:
		rows, err := r.q().ListBookingsPaged(ctx, db.ListBookingsPagedParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		return mapBookingRows(rows, err)
	}
}

func mapBookingRows(rows []db.Booking, err error) ([]Booking, error) {
	if err != nil {
		return nil, err
	}
	out := make([]Booking, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToBooking(row))
	}
	return out, nil
}

func (r *pgRepo) Count(ctx context.Context, villaSlug *string, status *Status) (int, error) {
	switch {
	case villaSlug != nil && status != nil:
		n, err := r.q().CountBookingsByVillaAndStatus(ctx, db.CountBookingsByVillaAndStatusParams{
			VillaSlug: *villaSlug,
			Status:    db.BookingStatus(*status),
		})
		return int(n), err
	case villaSlug != nil:
		n, err := r.q().CountBookingsByVilla(ctx, *villaSlug)
		return int(n), err
	case status != nil:
		n, err := r.q().CountBookingsByStatus(ctx, db.BookingStatus(*status))
		return int(n), err
	default:
		n, err := r.q().CountBookings(ctx)
		return int(n), err
	}
}

func (r *pgRepo) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("booking: invalid id: %w", err)
	}
	rows, err := r.q().DeleteBooking(ctx, pgtype.UUID{Bytes: [16]byte(uid), Valid: true})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
