//go:build integration

package booking

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

func TestPgRepo_SaveAndFindOverlap(t *testing.T) {
	pool := setupPg(t)
	repo := NewPgRepo(pool)
	ctx := context.Background()

	b, err := NewBooking(NewBookingInput{
		VillaSlug: "casadana", GuestName: "Jane", GuestEmail: "jane@example.com",
		CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08"),
		Adults: 2, Now: d("2026-05-12"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, b); err != nil {
		t.Fatalf("save: %v", err)
	}

	overlapping, err := repo.FindOverlapping(ctx, "casadana", d("2026-07-05"), d("2026-07-10"))
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(overlapping) != 1 {
		t.Errorf("overlapping = %d, want 1", len(overlapping))
	}
}

func TestPgRepo_ListFiltersByVillaSlug(t *testing.T) {
	pool := setupPg(t)
	repo := NewPgRepo(pool)
	ctx := context.Background()

	for _, slug := range []string{"casadana", "casacasay"} {
		b, err := NewBooking(NewBookingInput{
			VillaSlug: slug, GuestName: "Jane", GuestEmail: "jane@example.com",
			CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08"),
			Adults: 2, Now: d("2026-05-12"),
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.Save(ctx, b); err != nil {
			t.Fatalf("save: %v", err)
		}
	}

	slug := "casadana"
	rows, err := repo.List(ctx, &slug, nil, 20, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 1 || rows[0].VillaSlug != "casadana" {
		t.Errorf("rows = %+v, want exactly 1 casadana booking", rows)
	}

	count, err := repo.Count(ctx, &slug, nil)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}
