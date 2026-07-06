//go:build integration

package adminauth

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

func TestPgRepo_SaveFindDelete(t *testing.T) {
	pool := setupPg(t)
	repo := NewPgRepo(pool)
	ctx := context.Background()

	u := &AdminUser{ID: "11111111-1111-1111-1111-111111111111", Email: "loan@casa-dana.com", PasswordHash: "hash"}
	if err := repo.Save(ctx, u); err != nil {
		t.Fatalf("save: %v", err)
	}

	found, err := repo.FindByEmail(ctx, "loan@casa-dana.com")
	if err != nil {
		t.Fatalf("find by email: %v", err)
	}
	if found.ID != u.ID {
		t.Errorf("ID = %q, want %q", found.ID, u.ID)
	}

	if err := repo.Delete(ctx, u.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.FindByID(ctx, u.ID); err != ErrNotFound {
		t.Fatalf("err after delete = %v, want ErrNotFound", err)
	}
}

func TestPgRepo_Save_DuplicateEmail(t *testing.T) {
	pool := setupPg(t)
	repo := NewPgRepo(pool)
	ctx := context.Background()

	a := &AdminUser{ID: "11111111-1111-1111-1111-111111111111", Email: "dup@casa-dana.com", PasswordHash: "hash"}
	b := &AdminUser{ID: "22222222-2222-2222-2222-222222222222", Email: "dup@casa-dana.com", PasswordHash: "hash2"}
	if err := repo.Save(ctx, a); err != nil {
		t.Fatalf("save a: %v", err)
	}
	if err := repo.Save(ctx, b); err != ErrEmailTaken {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
}
