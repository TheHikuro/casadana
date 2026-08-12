//go:build integration

package pricing

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	dbpkg "github.com/TheHikuro/casadana/internal/db"
	pg "github.com/TheHikuro/casadana/internal/platform/postgres"
)

func setupPg(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("casadana_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("dsn: %v", err)
	}

	pool, err := pg.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pg.MigrateUp(pool, dbpkg.Migrations, "migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

func TestPgRepo_ListOverrides(t *testing.T) {
	pool := setupPg(t)
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO price_overrides (villa_slug, date, price_cents) VALUES
		 ('casadana', '2026-07-04', 25000),
		 ('casadana', '2026-07-05', 25000),
		 ('casacasay', '2026-07-04', 18000)`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	repo := NewPgRepo(pool)
	out, err := repo.ListOverrides(ctx, "casadana", d("2026-07-01"), d("2026-08-01"))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("len = %d, want 2", len(out))
	}
	if out[0].PriceCents != 25000 {
		t.Errorf("price_cents = %d, want 25000", out[0].PriceCents)
	}
}

func TestPgRepo_GetSettings_NeverConfigured(t *testing.T) {
	pool := setupPg(t)
	repo := NewPgRepo(pool)

	got, err := repo.GetSettings(context.Background(), "casadana")
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if got != (Settings{VillaSlug: "casadana"}) {
		t.Errorf("settings = %+v, want zero value", got)
	}
}

func TestPgRepo_SaveAndGetSettings(t *testing.T) {
	pool := setupPg(t)
	ctx := context.Background()
	repo := NewPgRepo(pool)

	in := Settings{VillaSlug: "casadana", BasePriceCents: 18500, MinNights: 3, CleaningFeeCents: 8000, ConciergeFeeCents: 5000}
	saved, err := repo.SaveSettings(ctx, in)
	if err != nil {
		t.Fatalf("save settings: %v", err)
	}
	if saved != in {
		t.Errorf("saved = %+v, want %+v", saved, in)
	}

	// Second write must update in place rather than conflict.
	in.BasePriceCents = 19500
	if _, err := repo.SaveSettings(ctx, in); err != nil {
		t.Fatalf("re-save settings: %v", err)
	}
	got, err := repo.GetSettings(ctx, "casadana")
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if got != in {
		t.Errorf("settings = %+v, want %+v", got, in)
	}
}

func TestPgRepo_SeasonRuleLifecycle(t *testing.T) {
	pool := setupPg(t)
	ctx := context.Background()
	repo := NewPgRepo(pool)

	rule, err := NewSeasonRule(NewSeasonRuleInput{
		VillaSlug: "casadana", Label: "Summer peak",
		Start: d("2026-07-01"), End: d("2026-08-31"), PriceCents: 25000,
	})
	if err != nil {
		t.Fatalf("new rule: %v", err)
	}
	inserted, err := repo.InsertSeasonRule(ctx, *rule)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if inserted.ID != rule.ID || !inserted.Start.Equal(d("2026-07-01")) {
		t.Errorf("inserted = %+v, want %+v", inserted, *rule)
	}

	inserted.PriceCents = 30000
	updated, err := repo.UpdateSeasonRule(ctx, inserted)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.PriceCents != 30000 || updated.Label != "Summer peak" {
		t.Errorf("updated = %+v", updated)
	}

	rules, err := repo.ListSeasonRules(ctx, "casadana")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("rules = %d, want 1", len(rules))
	}

	if err := repo.DeleteSeasonRule(ctx, rule.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.GetSeasonRule(ctx, rule.ID); err != ErrRuleNotFound {
		t.Fatalf("get after delete = %v, want ErrRuleNotFound", err)
	}
	if err := repo.DeleteSeasonRule(ctx, rule.ID); err != ErrRuleNotFound {
		t.Fatalf("double delete = %v, want ErrRuleNotFound", err)
	}
}
