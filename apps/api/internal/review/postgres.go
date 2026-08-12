package review

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	// Admin-authored reviews have no booking behind them: booking_id is NULL.
	var bookingID pgtype.UUID
	if rv.BookingID != "" {
		bid, err := uuid.Parse(rv.BookingID)
		if err != nil {
			return fmt.Errorf("review: invalid booking id: %w", err)
		}
		bookingID = pgtype.UUID{Bytes: [16]byte(bid), Valid: true}
	}
	_, err = r.q().InsertReview(ctx, db.InsertReviewParams{
		ID:         pgtype.UUID{Bytes: [16]byte(id), Valid: true},
		BookingID:  bookingID,
		VillaSlug:  rv.VillaSlug,
		AuthorName: rv.AuthorName,
		Rating:     int16(rv.Rating),
		Body:       rv.Body,
		Status:     db.ReviewStatus(rv.Status),
		Meta:       rv.Meta,
		Source:     rv.Source,
		Featured:   rv.Featured,
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

func (r *pgRepo) ListByVillaAndStatus(ctx context.Context, slug string, status *Status) ([]Review, error) {
	var dbStatus *db.ReviewStatus
	if status != nil {
		s := db.ReviewStatus(*status)
		dbStatus = &s
	}
	rows, err := r.q().ListReviewsByVillaAndStatus(ctx, db.ListReviewsByVillaAndStatusParams{
		VillaSlug: slug,
		Status:    dbStatus,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Review, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToReview(row))
	}
	return out, nil
}

func (r *pgRepo) Get(ctx context.Context, id string) (*Review, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("review: invalid id: %w", err)
	}
	row, err := r.q().GetReview(ctx, pgtype.UUID{Bytes: [16]byte(uid), Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	rv := rowToReview(row)
	return &rv, nil
}

func (r *pgRepo) Update(ctx context.Context, id string, patch UpdatePatch) (*Review, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("review: invalid id: %w", err)
	}
	params := db.UpdateReviewParams{
		ID:       pgtype.UUID{Bytes: [16]byte(uid), Valid: true},
		Featured: patch.Featured,
		Meta:     patch.Meta,
		Source:   patch.Source,
		Body:     patch.Body,
	}
	if patch.Status != nil {
		s := db.ReviewStatus(*patch.Status)
		params.Status = &s
	}
	if patch.Rating != nil {
		rating := int16(*patch.Rating)
		params.Rating = &rating
	}
	row, err := r.q().UpdateReview(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	rv := rowToReview(row)
	return &rv, nil
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

func (r *pgRepo) GetMeta(ctx context.Context, slug string) (ReviewMeta, error) {
	row, err := r.q().GetReviewMeta(ctx, slug)
	if err != nil {
		// A villa nobody has curated yet simply has no numbers to show.
		if errors.Is(err, pgx.ErrNoRows) {
			return ReviewMeta{VillaSlug: slug}, nil
		}
		return ReviewMeta{}, err
	}
	return rowToMeta(row), nil
}

func (r *pgRepo) UpsertMeta(ctx context.Context, m ReviewMeta) (ReviewMeta, error) {
	row, err := r.q().UpsertReviewMeta(ctx, db.UpsertReviewMetaParams{
		VillaSlug:    m.VillaSlug,
		DisplayAvg:   numericFromFloat(m.DisplayAvg),
		DisplayCount: int32(m.DisplayCount),
		Cleanliness:  numericFromFloat(m.Breakdown.Cleanliness),
		Comfort:      numericFromFloat(m.Breakdown.Comfort),
		Location:     numericFromFloat(m.Breakdown.Location),
		Host:         numericFromFloat(m.Breakdown.Host),
		Value:        numericFromFloat(m.Breakdown.Value),
	})
	if err != nil {
		return ReviewMeta{}, err
	}
	return rowToMeta(row), nil
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
		Meta:       row.Meta,
		Source:     row.Source,
		Featured:   row.Featured,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}
}

func rowToMeta(row db.VillaReviewMetum) ReviewMeta {
	return ReviewMeta{
		VillaSlug:    row.VillaSlug,
		DisplayAvg:   numericToFloat(row.DisplayAvg),
		DisplayCount: int(row.DisplayCount),
		Breakdown: Breakdown{
			Cleanliness: numericToFloat(row.Cleanliness),
			Comfort:     numericToFloat(row.Comfort),
			Location:    numericToFloat(row.Location),
			Host:        numericToFloat(row.Host),
			Value:       numericToFloat(row.Value),
		},
	}
}
