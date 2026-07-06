package adminauth

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

func (r *pgRepo) Save(ctx context.Context, u *AdminUser) error {
	id, err := uuid.Parse(u.ID)
	if err != nil {
		return fmt.Errorf("adminauth: invalid id: %w", err)
	}
	row, err := r.q().InsertAdminUser(ctx, db.InsertAdminUserParams{
		ID:           pgtype.UUID{Bytes: [16]byte(id), Valid: true},
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrEmailTaken
		}
		return err
	}
	u.CreatedAt = row.CreatedAt.Time
	return nil
}

func (r *pgRepo) FindByEmail(ctx context.Context, email string) (*AdminUser, error) {
	row, err := r.q().GetAdminUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	u := rowToAdminUser(row)
	return &u, nil
}

func (r *pgRepo) FindByID(ctx context.Context, id string) (*AdminUser, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("adminauth: invalid id: %w", err)
	}
	row, err := r.q().GetAdminUserByID(ctx, pgtype.UUID{Bytes: [16]byte(uid), Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	u := rowToAdminUser(row)
	return &u, nil
}

func (r *pgRepo) List(ctx context.Context) ([]AdminUser, error) {
	rows, err := r.q().ListAdminUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AdminUser, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToAdminUser(row))
	}
	return out, nil
}

func (r *pgRepo) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("adminauth: invalid id: %w", err)
	}
	rows, err := r.q().DeleteAdminUser(ctx, pgtype.UUID{Bytes: [16]byte(uid), Valid: true})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func rowToAdminUser(row db.AdminUser) AdminUser {
	idStr := ""
	if row.ID.Valid {
		u, _ := uuid.FromBytes(row.ID.Bytes[:])
		idStr = u.String()
	}
	return AdminUser{
		ID:           idStr,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		CreatedAt:    row.CreatedAt.Time,
	}
}
