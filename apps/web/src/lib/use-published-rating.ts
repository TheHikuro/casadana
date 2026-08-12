import { useGetVillaReviewMeta } from "@casa-dana/api"

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
 * Returns null while the query is in flight and for a villa with no approved
 * review: there is no invented figure to stand in, so callers drop their rating
 * row entirely rather than quote a score no guest ever gave.
 */
export function usePublishedRating(villaSlug: string): PublishedRating | null {
  const { data: meta } = useGetVillaReviewMeta(villaSlug)
  if (!meta || meta.display_count === 0) return null

  return {
    score: meta.display_avg,
    count: meta.display_count,
    filledStars: Math.min(5, Math.max(0, Math.round(meta.display_avg))),
  }
}
