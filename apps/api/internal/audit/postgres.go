package audit

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TheHikuro/casadana/internal/db"
)

type pgRepo struct {
	pool *pgxpool.Pool
}

func NewPgRepo(pool *pgxpool.Pool) Repository { return &pgRepo{pool: pool} }

func (r *pgRepo) q() *db.Queries { return db.New(r.pool) }

func (r *pgRepo) Save(ctx context.Context, e *Event) error {
	id, err := uuid.Parse(e.ID)
	if err != nil {
		return fmt.Errorf("audit: invalid id: %w", err)
	}
	row, err := r.q().InsertAuditEvent(ctx, db.InsertAuditEventParams{
		ID:         pgtype.UUID{Bytes: [16]byte(id), Valid: true},
		VillaSlug:  e.VillaSlug,
		Type:       db.AuditEventType(e.Type),
		Message:    e.Message,
		ActorEmail: e.ActorEmail,
	})
	if err != nil {
		return err
	}
	e.CreatedAt = row.CreatedAt.Time
	return nil
}

func (r *pgRepo) List(ctx context.Context, villaSlug string, limit, offset int) ([]Event, error) {
	rows, err := r.q().ListAuditEvents(ctx, db.ListAuditEventsParams{
		VillaSlug: villaSlug,
		Off:       int32(offset),
		Lim:       int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]Event, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToEvent(row))
	}
	return out, nil
}

func (r *pgRepo) Count(ctx context.Context, villaSlug string) (int, error) {
	n, err := r.q().CountAuditEvents(ctx, villaSlug)
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

func rowToEvent(row db.AuditEvent) Event {
	idStr := ""
	if row.ID.Valid {
		u, _ := uuid.FromBytes(row.ID.Bytes[:])
		idStr = u.String()
	}
	return Event{
		ID:         idStr,
		VillaSlug:  row.VillaSlug,
		Type:       EventType(row.Type),
		Message:    row.Message,
		ActorEmail: row.ActorEmail,
		CreatedAt:  row.CreatedAt.Time,
	}
}
