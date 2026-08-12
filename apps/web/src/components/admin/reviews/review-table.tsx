import {
  type Review,
  type ReviewStatus,
  getListAdminReviewsQueryKey,
  getListVillaReviewsQueryKey,
  useDeleteReview,
  usePatchReview,
} from "@casa-dana/api"
import { useQueryClient } from "@tanstack/react-query"
import { Trash2 } from "lucide-react"

import { REVIEW_STATUS_LABELS, REVIEW_STATUSES } from "@/components/admin/reviews/review-status"
import { EmptyState } from "@/components/admin/ui/empty-state"
import { Toggle } from "@/components/admin/ui/toggle"
import { useToast } from "@/components/ui/toast"

interface ReviewTableProps {
  reviews: Array<Review>
  property: "casadana" | "casacasay"
}

export default function ReviewTable({ reviews, property }: ReviewTableProps) {
  const queryClient = useQueryClient()
  const { toast } = useToast()

  // Invalidate at the admin endpoint prefix (the list is fetched unpaginated,
  // but the prefix also covers the per-status variants), plus the public list
  // this villa's site page reads from.
  const invalidateReviews = () => {
    queryClient.invalidateQueries({ queryKey: getListAdminReviewsQueryKey() })
    queryClient.invalidateQueries({ queryKey: getListVillaReviewsQueryKey(property) })
  }

  const { mutate: patchReview } = usePatchReview({
    mutation: {
      onSuccess: () => {
        invalidateReviews()
      },
      onError: () => toast("Could not update review"),
    },
  })

  const { mutate: deleteReview } = useDeleteReview({
    mutation: {
      onSuccess: () => {
        invalidateReviews()
        toast("Review deleted")
      },
      onError: () => toast("Could not delete review"),
    },
  })

  // The per-call onSuccess runs on top of the hook's own, which does the
  // invalidating — only the wording differs between the two patches.
  const handleFeatured = (r: Review, featured: boolean) => {
    patchReview(
      { id: r.id, data: { featured } },
      { onSuccess: () => toast(featured ? "Review featured" : "Review unfeatured") },
    )
  }

  const handleStatus = (r: Review, status: ReviewStatus) => {
    patchReview({ id: r.id, data: { status } }, { onSuccess: () => toast("Review status updated") })
  }

  const handleDelete = (r: Review) => {
    if (window.confirm(`Delete the review from ${r.author_name}?`)) {
      deleteReview({ id: r.id })
    }
  }

  if (reviews.length === 0) {
    return <EmptyState message="No reviews yet." />
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[720px] border-collapse text-[13px]">
        <thead>
          <tr className="border-outline-variant bg-surface-container-low text-on-surface-variant border-b text-left text-[10.5px] font-semibold tracking-[0.08em] uppercase">
            <th className="px-5 py-2.5">Guest</th>
            <th className="px-5 py-2.5">Rating</th>
            <th className="px-5 py-2.5">Quote</th>
            <th className="px-5 py-2.5">Featured</th>
            <th className="px-5 py-2.5">Status</th>
            <th className="px-5 py-2.5" />
          </tr>
        </thead>
        <tbody>
          {reviews.map((r) => (
            <tr key={r.id} className="border-outline-variant border-b last:border-0">
              <td className="px-5 py-3 align-top">
                <p className="text-on-surface font-semibold">{r.author_name}</p>
                {r.meta && <p className="text-on-surface-variant mt-0.5 text-[11.5px]">{r.meta}</p>}
              </td>
              <td className="px-5 py-3 align-top">
                <Stars rating={r.rating} />
              </td>
              <td className="text-on-surface-variant max-w-[340px] px-5 py-3 align-top text-[12.5px]">
                <p className="line-clamp-3">{r.body}</p>
              </td>
              <td className="px-5 py-3 align-top">
                <Toggle
                  checked={r.featured}
                  onChange={(next) => handleFeatured(r, next)}
                  label={`Feature the review from ${r.author_name}`}
                />
              </td>
              <td className="px-5 py-3 align-top">
                <select
                  value={r.status}
                  aria-label={`Status of the review from ${r.author_name}`}
                  onChange={(e) => handleStatus(r, e.target.value as ReviewStatus)}
                  className="border-outline-variant text-on-surface h-8 rounded-lg border bg-transparent px-2.5 text-[12.5px]"
                >
                  {REVIEW_STATUSES.map((s) => (
                    <option key={s} value={s}>
                      {REVIEW_STATUS_LABELS[s]}
                    </option>
                  ))}
                </select>
              </td>
              <td className="px-5 py-3 align-top">
                <button
                  type="button"
                  onClick={() => handleDelete(r)}
                  aria-label={`Delete the review from ${r.author_name}`}
                  className="text-on-surface-variant hover:bg-error-container hover:text-on-error-container rounded-md p-1.5"
                >
                  <Trash2 className="size-3.5" />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function Stars({ rating }: { rating: number }) {
  const filled = Math.max(0, Math.min(5, Math.round(rating)))
  return (
    <span className="whitespace-nowrap" aria-label={`${filled} out of 5`}>
      <span className="text-on-surface">{"★".repeat(filled)}</span>
      <span className="text-on-surface-variant/50">{"★".repeat(5 - filled)}</span>
    </span>
  )
}
