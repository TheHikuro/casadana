package pricing

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TheHikuro/casadana/internal/db"
	pg "github.com/TheHikuro/casadana/internal/platform/postgres"
)

type pgRepo struct {
	pool *pgxpool.Pool
}

func NewPgRepo(pool *pgxpool.Pool) Repository { return &pgRepo{pool: pool} }

func (r *pgRepo) q() *db.Queries { return db.New(r.pool) }

func (r *pgRepo) ListOverrides(ctx context.Context, villaSlug string, from, to time.Time) ([]PriceOverride, error) {
	rows, err := r.q().ListPriceOverrides(ctx, db.ListPriceOverridesParams{
		VillaSlug: villaSlug,
		From:      pgtype.Date{Time: from, Valid: true},
		To:        pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	out := make([]PriceOverride, 0, len(rows))
	for _, row := range rows {
		out = append(out, PriceOverride{
			VillaSlug:  row.VillaSlug,
			Date:       row.Date.Time,
			PriceCents: int(row.PriceCents),
		})
	}
	return out, nil
}

func (r *pgRepo) UpsertMany(ctx context.Context, villaSlug string, priceCents int, dates []time.Time) error {
	return pg.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		for _, dt := range dates {
			if err := q.UpsertPriceOverride(ctx, db.UpsertPriceOverrideParams{
				VillaSlug:  villaSlug,
				Date:       pgtype.Date{Time: dt, Valid: true},
				PriceCents: int32(priceCents),
			}); err != nil {
				return fmt.Errorf("upsert %s: %w", dt.Format("2006-01-02"), err)
			}
		}
		return nil
	})
}
