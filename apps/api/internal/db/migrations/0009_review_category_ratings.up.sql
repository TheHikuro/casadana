-- Per-category scores for a single review. Nullable throughout: a guest
-- submitting through our own form only leaves an overall rating, and a review
-- transcribed from another platform may not list every category. NULLs drop out
-- of an AVG on their own, so a category ends up averaged over exactly the
-- reviews that actually scored it.
ALTER TABLE reviews
    ADD COLUMN rating_cleanliness NUMERIC(3, 2) CHECK (rating_cleanliness BETWEEN 1 AND 5),
    ADD COLUMN rating_comfort     NUMERIC(3, 2) CHECK (rating_comfort     BETWEEN 1 AND 5),
    ADD COLUMN rating_location    NUMERIC(3, 2) CHECK (rating_location    BETWEEN 1 AND 5),
    ADD COLUMN rating_host        NUMERIC(3, 2) CHECK (rating_host        BETWEEN 1 AND 5),
    ADD COLUMN rating_value       NUMERIC(3, 2) CHECK (rating_value       BETWEEN 1 AND 5);

-- The published rating is now derived from the approved reviews themselves —
-- approving a review makes it count, un-approving it takes it back out — so the
-- hand-curated figures this table held have no reader left. Dropping it is what
-- keeps the public numbers from silently disagreeing with the reviews on show.
DROP TABLE villa_review_meta;
