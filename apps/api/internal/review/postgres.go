package review

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TheHikuro/casadana/internal/db"
)

type pgRepo struct {
	pool *pgxpool.Pool
}

func NewPgRepo(pool *pgxpool.Pool) Repository { return &pgRepo{pool: pool} }

func (r *pgRepo) q() *db.Queries { return db.New(r.pool) }

func (r *pgRepo) Save(ctx context.Context, rv *Review) error {
	id, err := uuid.Parse(rv.ID)
	if err != nil {
		return fmt.Errorf("review: invalid id: %w", err)
	}
	bookingID, err := uuid.Parse(rv.BookingID)
	if err != nil {
		return fmt.Errorf("review: invalid booking id: %w", err)
	}
	_, err = r.q().InsertReview(ctx, db.InsertReviewParams{
		ID:         pgtype.UUID{Bytes: [16]byte(id), Valid: true},
		BookingID:  pgtype.UUID{Bytes: [16]byte(bookingID), Valid: true},
		VillaSlug:  rv.VillaSlug,
		AuthorName: rv.AuthorName,
		Rating:     int16(rv.Rating),
		Body:       rv.Body,
		Status:     db.ReviewStatus(rv.Status),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyReviewed
		}
		return err
	}
	return nil
}

func (r *pgRepo) ListByVillaSlug(ctx context.Context, slug string) ([]Review, error) {
	rows, err := r.q().ListReviewsByVilla(ctx, slug)
	if err != nil {
		return nil, err
	}
	out := make([]Review, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToReview(row))
	}
	return out, nil
}

func (r *pgRepo) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("review: invalid id: %w", err)
	}
	rows, err := r.q().DeleteReview(ctx, pgtype.UUID{Bytes: [16]byte(uid), Valid: true})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func rowToReview(row db.Review) Review {
	idStr := ""
	if row.ID.Valid {
		u, _ := uuid.FromBytes(row.ID.Bytes[:])
		idStr = u.String()
	}
	bookingIDStr := ""
	if row.BookingID.Valid {
		u, _ := uuid.FromBytes(row.BookingID.Bytes[:])
		bookingIDStr = u.String()
	}
	return Review{
		ID:         idStr,
		BookingID:  bookingIDStr,
		VillaSlug:  row.VillaSlug,
		AuthorName: row.AuthorName,
		Rating:     int(row.Rating),
		Body:       row.Body,
		Status:     Status(row.Status),
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}
}
