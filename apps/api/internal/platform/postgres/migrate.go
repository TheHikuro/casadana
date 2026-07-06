package postgres

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

// MigrateUp runs all pending migrations from the given embedded filesystem.
// subPath is the directory within the FS that contains the *.sql files
// (e.g. "migrations" if the FS is rooted at internal/db/).
func MigrateUp(pool *pgxpool.Pool, migrationsFS fs.FS, subPath string) error {
	// Closing sqlDB does NOT close the underlying *pgxpool.Pool — per
	// stdlib.OpenDBFromPool's doc comment, closing the *sql.DB it returns only
	// releases the pgx connection(s) it borrowed back to the pool; it never
	// touches the pool itself. We deliberately do not close sqlDB here (it's
	// cheap to leave around with MaxIdleConns(0)), but we MUST close the
	// migrate driver below: golang-migrate's postgres driver acquires and
	// holds a single dedicated *sql.Conn for its lifetime (see WithInstance),
	// only releasing it in Close(). Without that Close, the borrowed pgx
	// connection stays permanently "Acquired" from the pool's perspective, and
	// any later pool.Close() (e.g. via t.Cleanup in tests) deadlocks forever
	// in puddle's destructWG.Wait().
	sqlDB := stdlib.OpenDBFromPool(pool)

	driver, err := migratepg.WithInstance(sqlDB, &migratepg.Config{})
	if err != nil {
		return fmt.Errorf("migrate: driver: %w", err)
	}

	src, err := iofs.New(migrationsFS, subPath)
	if err != nil {
		return fmt.Errorf("migrate: iofs source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate: new: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate: up: %w", err)
	}
	return nil
}
