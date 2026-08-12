import { useGetVillaReviewMeta } from "@casa-dana/api"

interface Fallback {
  score: number
  count: number
}

export interface PublishedRating {
  score: number
  count: number
  /** Filled star glyphs to draw, so a star row can't overstate the score. */
  filledStars: number
}

/**
 * A villa's published rating, computed by the API from its approved reviews —
 * the same figures the reviews section shows, so no two places on a page can
 * quote different numbers.
 *
 * `display_avg` is 0 for a villa with no approved review, which is why the
 * fallback is gated on `display_count` rather than on truthiness: the static
 * showcase figures in the content constants stand in while the query is in
 * flight and until the first review is approved, so a marketing page never
 * reads "0.00 / 5".
 */
export function usePublishedRating(villaSlug: string, fallback: Fallback): PublishedRating {
  const { data: meta } = useGetVillaReviewMeta(villaSlug)
  const published = meta && meta.display_count > 0 ? meta : null
  const score = published ? published.display_avg : fallback.score

  return {
    score,
    count: published ? published.display_count : fallback.count,
    filledStars: Math.min(5, Math.max(0, Math.round(score))),
  }
}
