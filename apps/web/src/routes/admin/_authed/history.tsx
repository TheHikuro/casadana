import { useListAdminHistory } from "@casa-dana/api"
import { keepPreviousData } from "@tanstack/react-query"
import { createFileRoute, useNavigate } from "@tanstack/react-router"

import HistoryTable from "@/components/admin/history/history-table"
import { AdminCard } from "@/components/admin/ui/admin-card"
import { EmptyState } from "@/components/admin/ui/empty-state"
import { Pager } from "@/components/admin/ui/pager"
import { StatRow, StatTile } from "@/components/admin/ui/stat-tile"

const PROPERTIES = ["casadana", "casacasay"] as const
type Property = (typeof PROPERTIES)[number]
// The activity log is denser and less scannable than the other screens, so it
// pages 20 at a time — the API's own default (max 100).
const PAGE_SIZE = 20

interface HistorySearch {
  property: Property
  page: number
}

function isProperty(value: unknown): value is Property {
  return typeof value === "string" && (PROPERTIES as ReadonlyArray<string>).includes(value)
}

function validateHistorySearch(search: Record<string, unknown>): HistorySearch {
  const property = isProperty(search.property) ? search.property : "casadana"
  const page = Number(search.page)
  return { property, page: Number.isFinite(page) && page >= 1 ? page : 1 }
}

export const Route = createFileRoute("/admin/_authed/history")({
  validateSearch: validateHistorySearch,
  component: HistoryPage,
})

const PROPERTY_LABELS: Record<Property, string> = {
  casadana: "Casa DaNa",
  casacasay: "Casa CasAy",
}

function HistoryPage() {
  const { property, page } = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })

  const { data, isPending, isError } = useListAdminHistory(
    { villa_slug: property, page, limit: PAGE_SIZE },
    { query: { placeholderData: keepPreviousData } },
  )
  const events = data?.events ?? []
  const maxPage = data ? Math.max(1, Math.ceil(data.total / PAGE_SIZE)) : 1

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
          <h2 className="text-on-surface text-2xl font-bold">History</h2>
          <p className="text-on-surface-variant mt-1 text-[13.5px]">
            A log of every change made in this dashboard.
          </p>
        </div>
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
      </div>

      <StatRow>
        <StatTile label="Total events" value={data?.total ?? 0} />
      </StatRow>

      <AdminCard
        title="Activity log"
        sub="Most recent first — every reservation, price, review and owner change."
        flush
      >
        {isError ? (
          <EmptyState message="Could not load the activity log." />
        ) : isPending ? (
          <EmptyState message="Loading activity…" />
        ) : events.length === 0 ? (
          <EmptyState message="No activity yet." />
        ) : (
          <HistoryTable events={events} />
        )}
        <Pager page={page} maxPage={maxPage} onPageChange={goToPage} />
      </AdminCard>
    </div>
  )
}
