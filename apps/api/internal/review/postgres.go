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
		ID:                pgtype.UUID{Bytes: [16]byte(id), Valid: true},
		BookingID:         bookingID,
		VillaSlug:         rv.VillaSlug,
		AuthorName:        rv.AuthorName,
		Rating:            int16(rv.Rating),
		Body:              rv.Body,
		Status:            db.ReviewStatus(rv.Status),
		Meta:              rv.Meta,
		Source:            rv.Source,
		Featured:          rv.Featured,
		RatingCleanliness: numericFromScore(rv.Categories.Cleanliness),
		RatingComfort:     numericFromScore(rv.Categories.Comfort),
		RatingLocation:    numericFromScore(rv.Categories.Location),
		RatingHost:        numericFromScore(rv.Categories.Host),
		RatingValue:       numericFromScore(rv.Categories.Value),
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
		ID:                pgtype.UUID{Bytes: [16]byte(uid), Valid: true},
		Featured:          patch.Featured,
		Meta:              patch.Meta,
		Source:            patch.Source,
		Body:              patch.Body,
		RatingCleanliness: numericFromScore(patch.Categories.Cleanliness),
		RatingComfort:     numericFromScore(patch.Categories.Comfort),
		RatingLocation:    numericFromScore(patch.Categories.Location),
		RatingHost:        numericFromScore(patch.Categories.Host),
		RatingValue:       numericFromScore(patch.Categories.Value),
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

func (r *pgRepo) GetAggregate(ctx context.Context, slug string) (ReviewMeta, error) {
	// An aggregate over no rows still returns one row — count 0 and NULL
	// averages — so a villa without approved reviews needs no special case.
	row, err := r.q().GetVillaReviewAggregate(ctx, slug)
	if err != nil {
		return ReviewMeta{}, err
	}
	meta := ReviewMeta{
		VillaSlug:    slug,
		DisplayCount: int(row.ReviewCount),
		Breakdown: Breakdown{
			Cleanliness: numericToScore(row.AvgCleanliness),
			Comfort:     numericToScore(row.AvgComfort),
			Location:    numericToScore(row.AvgLocation),
			Host:        numericToScore(row.AvgHost),
			Value:       numericToScore(row.AvgValue),
		},
	}
	// The overall average is the headline figure, so it reads as a plain 0 when
	// there is nothing to average rather than as an absent value.
	if avg := numericToScore(row.AvgRating); avg != nil {
		meta.DisplayAvg = *avg
	}
	return meta, nil
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
		Categories: CategoryRatings{
			Cleanliness: numericToScore(row.RatingCleanliness),
			Comfort:     numericToScore(row.RatingComfort),
			Location:    numericToScore(row.RatingLocation),
			Host:        numericToScore(row.RatingHost),
			Value:       numericToScore(row.RatingValue),
		},
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}
