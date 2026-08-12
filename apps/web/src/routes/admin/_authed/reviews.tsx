import { useListAdminReviews } from "@casa-dana/api"
import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { useEffect } from "react"

import AddReviewDialog from "@/components/admin/reviews/add-review-dialog"
import RatingBreakdownCard from "@/components/admin/reviews/rating-breakdown-card"
import ReviewTable from "@/components/admin/reviews/review-table"
import { AdminCard } from "@/components/admin/ui/admin-card"
import { Pager } from "@/components/admin/ui/pager"
import { StatRow, StatTile } from "@/components/admin/ui/stat-tile"

const PROPERTIES = ["casadana", "casacasay"] as const
type Property = (typeof PROPERTIES)[number]
const PAGE_SIZE = 8

interface ReviewsSearch {
  property: Property
  page: number
}

function isProperty(value: unknown): value is Property {
  return typeof value === "string" && (PROPERTIES as ReadonlyArray<string>).includes(value)
}

function validateReviewsSearch(search: Record<string, unknown>): ReviewsSearch {
  const property = isProperty(search.property) ? search.property : "casadana"
  const page = Number(search.page)
  return { property, page: Number.isFinite(page) && page >= 1 ? page : 1 }
}

export const Route = createFileRoute("/admin/_authed/reviews")({
  validateSearch: validateReviewsSearch,
  component: ReviewsPage,
})

const PROPERTY_LABELS: Record<Property, string> = {
  casadana: "Casa DaNa",
  casacasay: "Casa CasAy",
}

function ReviewsPage() {
  const { property, page } = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })

  // The admin endpoint returns every review for the villa in one go, so the
  // stats and the pagination below are both derived client-side.
  const { data } = useListAdminReviews({ villa_slug: property })
  const reviews = data?.reviews ?? []

  const total = reviews.length
  const published = reviews.filter((r) => r.status === "approved").length
  const featured = reviews.filter((r) => r.featured).length
  const average = total === 0 ? 0 : reviews.reduce((sum, r) => sum + r.rating, 0) / total

  const maxPage = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const currentPage = Math.min(page, maxPage)
  const visible = reviews.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE)

  // Deleting the last row of the last page would otherwise strand the URL on a
  // page that no longer exists.
  useEffect(() => {
    if (page > maxPage) {
      navigate({ replace: true, search: (prev) => ({ ...prev, page: maxPage }) })
    }
  }, [page, maxPage, navigate])

  const goToPage = (nextPage: number) => {
    navigate({ search: (prev) => ({ ...prev, page: nextPage }) })
  }

  const switchProperty = (nextProperty: Property) => {
    navigate({ search: (prev) => ({ ...prev, property: nextProperty, page: 1 }) })
  }

  return (
    <div>
      <div className="mb-7 flex flex-wrap items-baseline justify-between gap-4">
        <div>
          <h2 className="text-on-surface text-2xl font-bold">Reviews</h2>
          <p className="text-on-surface-variant mt-1 text-[13.5px]">
            Guest testimonials and the rating figures shown for {PROPERTY_LABELS[property]}.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <div className="bg-surface-container flex gap-1 rounded-lg p-1">
            {PROPERTIES.map((p) => (
              <button
                key={p}
                type="button"
                onClick={() => switchProperty(p)}
                className={
                  p === property
                    ? "bg-primary text-on-primary rounded-md px-3 py-1.5 text-[13px] font-medium"
                    : "text-on-surface-variant rounded-md px-3 py-1.5 text-[13px] font-medium"
                }
              >
                {PROPERTY_LABELS[p]}
              </button>
            ))}
          </div>
          <AddReviewDialog property={property} />
        </div>
      </div>

      <StatRow>
        <StatTile label="Average rating" value={average.toFixed(2)} />
        <StatTile label="Total reviews" value={total} />
        <StatTile label="Published" value={published} />
        <StatTile label="Featured" value={featured} />
      </StatRow>

      <div className="flex flex-col gap-5">
        {/* Remounted on a property switch so the form re-seeds from that villa's meta. */}
        <RatingBreakdownCard key={property} property={property} />

        <AdminCard title="Individual reviews" flush>
          <ReviewTable reviews={visible} property={property} />
          <Pager page={currentPage} maxPage={maxPage} onPageChange={goToPage} />
        </AdminCard>
      </div>
    </div>
  )
}
