-- Append-only activity log backing the admin History screen. Every mutating
-- admin action records one row; nothing ever updates or deletes them.
CREATE TYPE audit_event_type AS ENUM ('reservation', 'pricing', 'review', 'owner', 'system');

CREATE TABLE audit_events (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    villa_slug  TEXT NOT NULL,
    type        audit_event_type NOT NULL,
    message     TEXT NOT NULL,
    actor_email TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX audit_events_villa_created_idx ON audit_events (villa_slug, created_at DESC);
