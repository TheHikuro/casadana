package pricing

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

func (r *pgRepo) GetSettings(ctx context.Context, villaSlug string) (Settings, error) {
	row, err := r.q().GetPricingSettings(ctx, villaSlug)
	if err != nil {
		// No row means "known villa, never configured" — the zero value is
		// the answer, not a not-found error.
		if errors.Is(err, pgx.ErrNoRows) {
			return Settings{VillaSlug: villaSlug}, nil
		}
		return Settings{}, err
	}
	return rowToSettings(row), nil
}

func (r *pgRepo) SaveSettings(ctx context.Context, s Settings) (Settings, error) {
	row, err := r.q().UpsertPricingSettings(ctx, db.UpsertPricingSettingsParams{
		VillaSlug:         s.VillaSlug,
		BasePriceCents:    int32(s.BasePriceCents),
		MinNights:         int16(s.MinNights),
		CleaningFeeCents:  int32(s.CleaningFeeCents),
		ConciergeFeeCents: int32(s.ConciergeFeeCents),
	})
	if err != nil {
		return Settings{}, err
	}
	return rowToSettings(row), nil
}

func (r *pgRepo) ListSeasonRules(ctx context.Context, villaSlug string) ([]SeasonRule, error) {
	rows, err := r.q().ListSeasonRules(ctx, villaSlug)
	if err != nil {
		return nil, err
	}
	out := make([]SeasonRule, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToSeasonRule(row))
	}
	return out, nil
}

func (r *pgRepo) GetSeasonRule(ctx context.Context, id string) (SeasonRule, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return SeasonRule{}, fmt.Errorf("pricing: invalid season rule id: %w", err)
	}
	row, err := r.q().GetSeasonRule(ctx, pgtype.UUID{Bytes: [16]byte(uid), Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SeasonRule{}, ErrRuleNotFound
		}
		return SeasonRule{}, err
	}
	return rowToSeasonRule(row), nil
}

func (r *pgRepo) InsertSeasonRule(ctx context.Context, rule SeasonRule) (SeasonRule, error) {
	uid, err := uuid.Parse(rule.ID)
	if err != nil {
		return SeasonRule{}, fmt.Errorf("pricing: invalid season rule id: %w", err)
	}
	row, err := r.q().InsertSeasonRule(ctx, db.InsertSeasonRuleParams{
		ID:         pgtype.UUID{Bytes: [16]byte(uid), Valid: true},
		VillaSlug:  rule.VillaSlug,
		Label:      rule.Label,
		StartDate:  pgtype.Date{Time: rule.Start, Valid: true},
		EndDate:    pgtype.Date{Time: rule.End, Valid: true},
		PriceCents: int32(rule.PriceCents),
	})
	if err != nil {
		return SeasonRule{}, err
	}
	return rowToSeasonRule(row), nil
}

// UpdateSeasonRule writes every column: the service has already merged the
// caller's partial patch onto the stored row and validated the result.
func (r *pgRepo) UpdateSeasonRule(ctx context.Context, rule SeasonRule) (SeasonRule, error) {
	uid, err := uuid.Parse(rule.ID)
	if err != nil {
		return SeasonRule{}, fmt.Errorf("pricing: invalid season rule id: %w", err)
	}
	priceCents := int32(rule.PriceCents)
	row, err := r.q().UpdateSeasonRule(ctx, db.UpdateSeasonRuleParams{
		ID:         pgtype.UUID{Bytes: [16]byte(uid), Valid: true},
		Label:      &rule.Label,
		StartDate:  pgtype.Date{Time: rule.Start, Valid: true},
		EndDate:    pgtype.Date{Time: rule.End, Valid: true},
		PriceCents: &priceCents,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SeasonRule{}, ErrRuleNotFound
		}
		return SeasonRule{}, err
	}
	return rowToSeasonRule(row), nil
}

func (r *pgRepo) DeleteSeasonRule(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("pricing: invalid season rule id: %w", err)
	}
	rows, err := r.q().DeleteSeasonRule(ctx, pgtype.UUID{Bytes: [16]byte(uid), Valid: true})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrRuleNotFound
	}
	return nil
}

func rowToSettings(row db.VillaPricingSetting) Settings {
	return Settings{
		VillaSlug:         row.VillaSlug,
		BasePriceCents:    int(row.BasePriceCents),
		MinNights:         int(row.MinNights),
		CleaningFeeCents:  int(row.CleaningFeeCents),
		ConciergeFeeCents: int(row.ConciergeFeeCents),
	}
}

func rowToSeasonRule(row db.SeasonRule) SeasonRule {
	idStr := ""
	if row.ID.Valid {
		u, _ := uuid.FromBytes(row.ID.Bytes[:])
		idStr = u.String()
	}
	return SeasonRule{
		ID:         idStr,
		VillaSlug:  row.VillaSlug,
		Label:      row.Label,
		Start:      row.StartDate.Time,
		End:        row.EndDate.Time,
		PriceCents: int(row.PriceCents),
	}
}
