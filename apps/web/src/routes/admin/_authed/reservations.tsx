import { keepPreviousData } from "@tanstack/react-query"
import { createFileRoute, useNavigate } from "@tanstack/react-router"

import { type BookingStatus, useListBookings } from "@casa-dana/api"
import AddReservationDialog from "@/components/admin/add-reservation-dialog"
import ReservationTable from "@/components/admin/reservation-table"
import { Button } from "@/components/ui/button"

const PROPERTIES = ["casadana", "casacasay"] as const
type Property = (typeof PROPERTIES)[number]
const STATUSES: Array<BookingStatus> = ["pending", "approved", "rejected", "cancelled", "paid"]
const PAGE_SIZE = 8

interface ReservationsSearch {
  property: Property
  status?: BookingStatus
  page: number
}

function isProperty(value: unknown): value is Property {
  return typeof value === "string" && (PROPERTIES as ReadonlyArray<string>).includes(value)
}

function isBookingStatus(value: unknown): value is BookingStatus {
  return typeof value === "string" && (STATUSES as ReadonlyArray<string>).includes(value)
}

function validateReservationsSearch(search: Record<string, unknown>): ReservationsSearch {
  const property = isProperty(search.property) ? search.property : "casadana"
  const status = isBookingStatus(search.status) ? search.status : undefined
  const page = Number(search.page)
  return { property, status, page: Number.isFinite(page) && page >= 1 ? page : 1 }
}

export const Route = createFileRoute("/admin/_authed/reservations")({
  validateSearch: validateReservationsSearch,
  component: ReservationsPage,
})

const PROPERTY_LABELS: Record<Property, string> = {
  casadana: "Casa DaNa",
  casacasay: "Casa CasAy",
}

function ReservationsPage() {
  const { property, status, page } = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })

  const { data } = useListBookings(
    { villa_slug: property, status, page, limit: PAGE_SIZE },
    { query: { placeholderData: keepPreviousData } },
  )
  const { data: totalsAll } = useListBookings({ villa_slug: property, limit: 1 })
  const statusTotals = useStatusTotals(property)
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
          <h2 className="text-2xl font-bold text-on-surface">Reservations</h2>
          <p className="mt-1 text-[13.5px] text-on-surface-variant">
            Requests and confirmed stays for {PROPERTY_LABELS[property]}.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex gap-1 rounded-lg bg-surface-container p-1">
            {PROPERTIES.map((p) => (
              <button
                key={p}
                type="button"
                onClick={() => switchProperty(p)}
                className={
                  p === property
                    ? "rounded-md bg-primary px-3 py-1.5 text-[13px] font-medium text-on-primary"
                    : "rounded-md px-3 py-1.5 text-[13px] font-medium text-on-surface-variant"
                }
              >
                {PROPERTY_LABELS[p]}
              </button>
            ))}
          </div>
          <AddReservationDialog property={property} status={status} page={page} />
        </div>
      </div>

      <div className="mb-5 grid grid-cols-2 gap-4 md:grid-cols-6">
        <StatTile label="Total" value={totalsAll?.total ?? 0} />
        {STATUSES.map((s) => (
          <StatTile key={s} label={s} value={statusTotals[s] ?? 0} />
        ))}
      </div>

      <div className="rounded-lg border border-outline-variant bg-surface">
        <div className="border-b border-outline-variant px-5 py-4">
          <h3 className="text-[14.5px] font-semibold text-on-surface">All reservations</h3>
        </div>
        <ReservationTable bookings={data?.bookings ?? []} property={property} status={status} page={page} />
        {data && data.total > PAGE_SIZE && (
          <div className="flex items-center justify-center gap-4 border-t border-outline-variant px-5 py-3.5">
            <Button type="button" variant="outline" size="sm" disabled={page <= 1} onClick={() => goToPage(page - 1)}>
              ‹ Prev
            </Button>
            <span className="text-[12.5px] text-on-surface-variant">
              Page {page} of {maxPage}
            </span>
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={page >= maxPage}
              onClick={() => goToPage(page + 1)}
            >
              Next ›
            </Button>
          </div>
        )}
      </div>
    </div>
  )
}

function useStatusTotals(property: Property): Partial<Record<BookingStatus, number>> {
  const pendingQuery = useListBookings({ villa_slug: property, status: "pending", limit: 1 })
  const approvedQuery = useListBookings({ villa_slug: property, status: "approved", limit: 1 })
  const rejectedQuery = useListBookings({ villa_slug: property, status: "rejected", limit: 1 })
  const cancelledQuery = useListBookings({ villa_slug: property, status: "cancelled", limit: 1 })
  const paidQuery = useListBookings({ villa_slug: property, status: "paid", limit: 1 })
  return {
    pending: pendingQuery.data?.total,
    approved: approvedQuery.data?.total,
    rejected: rejectedQuery.data?.total,
    cancelled: cancelledQuery.data?.total,
    paid: paidQuery.data?.total,
  }
}

function StatTile({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border border-outline-variant bg-surface px-4 py-3.5">
      <p className="text-[11px] font-semibold tracking-[0.06em] text-on-surface-variant uppercase">{label}</p>
      <p className="mt-1.5 font-mono text-2xl font-bold text-on-surface">{value}</p>
    </div>
  )
}
