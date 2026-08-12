DROP TABLE IF EXISTS villa_review_meta;

DROP INDEX IF EXISTS reviews_featured_idx;

ALTER TABLE reviews
    DROP COLUMN IF EXISTS source,
    DROP COLUMN IF EXISTS meta,
    DROP COLUMN IF EXISTS featured;

DELETE FROM reviews WHERE booking_id IS NULL;
ALTER TABLE reviews ALTER COLUMN booking_id SET NOT NULL;
