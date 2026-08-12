-- The locale the guest browsed the site in, so transactional mail about this
-- booking can be written in their language rather than the site's source one.
-- Rows created before this column existed default to French, which is what
-- they were in fact emailed in.
ALTER TABLE bookings ADD COLUMN locale TEXT NOT NULL DEFAULT 'fr'
    CONSTRAINT bookings_locale_valid CHECK (locale IN ('fr', 'en', 'es'));
