import {
  type BookingResponse,
  type BookingStatus,
  type PatchBookingRequestStatus,
  getListBookingsQueryKey,
  useDeleteBooking,
  useListBookings,
  usePatchBooking,
} from "@casa-dana/api"
import { useQueryClient } from "@tanstack/react-query"
import { Trash2 } from "lucide-react"

import { Button } from "@/components/ui/button"
import { useToast } from "@/components/ui/toast"

// Half-open interval overlap, matching the backend's own conflict check
// (check_in < other.check_out AND check_out > other.check_in). ISO
// "YYYY-MM-DD" strings compare correctly with plain string operators.
function datesOverlap(aIn: string, aOut: string, bIn: string, bOut: string): boolean {
  return aIn < bOut && aOut > bIn
}

const NEXT_STATUSES: Record<
  BookingStatus,
  Array<{ status: PatchBookingRequestStatus; label: string }>
> = {
  pending: [
    { status: "approved", label: "Approve" },
    { status: "rejected", label: "Reject" },
    { status: "cancelled", label: "Cancel" },
  ],
  approved: [
    { status: "paid", label: "Mark paid" },
    { status: "cancelled", label: "Cancel" },
  ],
  paid: [{ status: "cancelled", label: "Cancel" }],
  rejected: [],
  cancelled: [],
}

const STATUS_BADGE_CLASSES: Record<BookingStatus, string> = {
  pending: "bg-secondary-container text-on-secondary-container",
  approved: "bg-primary-container text-on-primary-container",
  paid: "bg-primary text-on-primary",
  rejected: "bg-error-container text-on-error-container",
  cancelled: "bg-surface-container-high text-on-surface-variant",
}

interface ReservationTableProps {
  bookings: Array<BookingResponse>
  property: "casadana" | "casacasay"
}

export default function ReservationTable({ bookings, property }: ReservationTableProps) {
  const queryClient = useQueryClient()
  const { toast } = useToast()
  // Invalidate at the endpoint prefix (not the exact page/limit key) so this
  // also refreshes the stat row's separate limit:1 queries, not just this page.
  const bookingsQueryKey = getListBookingsQueryKey()

  // Fetched independently of pagination so "Approve" is disabled for a
  // conflict even when the confirmed booking it collides with sits on a
  // different page. Mirrors the backend's own approve-time conflict check
  // (booking/service.go's TransitionStatus) as a proactive UI hint — the
  // backend's check is the authoritative guard either way.
  const { data: approvedData } = useListBookings({
    villa_slug: property,
    status: "approved",
    limit: 100,
  })
  const { data: paidData } = useListBookings({ villa_slug: property, status: "paid", limit: 100 })
  const confirmedBookings = [...(approvedData?.bookings ?? []), ...(paidData?.bookings ?? [])]

  const conflictingApproval = (b: BookingResponse): boolean =>
    confirmedBookings.some(
      (other) =>
        other.id !== b.id && datesOverlap(b.check_in, b.check_out, other.check_in, other.check_out),
    )

  const { mutate: patchStatus } = usePatchBooking({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: bookingsQueryKey })
        toast("Status updated")
      },
      onError: () => toast("Could not update status"),
    },
  })

  const { mutate: deleteBooking } = useDeleteBooking({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: bookingsQueryKey })
        toast("Reservation deleted")
      },
      onError: () => toast("Could not delete reservation"),
    },
  })

  const handleDelete = (id: string) => {
    if (window.confirm("Delete this reservation?")) {
      deleteBooking({ id })
    }
  }

  if (bookings.length === 0) {
    return (
      <div className="text-on-surface-variant px-5 py-10 text-center text-[13.5px]">
        No reservations yet — new "Request to Book" submissions from the public site will land here.
      </div>
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[720px] border-collapse text-[13px]">
        <thead>
          <tr className="border-outline-variant bg-surface-container-low text-on-surface-variant border-b text-left text-[10.5px] font-semibold tracking-[0.08em] uppercase">
            <th className="px-5 py-2.5">Guest</th>
            <th className="px-5 py-2.5">Dates</th>
            <th className="px-5 py-2.5">Guests</th>
            <th className="px-5 py-2.5">Source</th>
            <th className="px-5 py-2.5">Status</th>
            <th className="px-5 py-2.5" />
          </tr>
        </thead>
        <tbody>
          {bookings.map((b) => (
            <tr key={b.id} className="border-outline-variant border-b last:border-0">
              <td className="px-5 py-3">
                <p className="text-on-surface font-semibold">{b.guest_name}</p>
                <p className="text-on-surface-variant mt-0.5 font-mono text-[11px]">
                  {b.guest_email}
                </p>
              </td>
              <td className="text-on-surface px-5 py-3 font-mono">
                {b.check_in} → {b.check_out}
              </td>
              <td className="text-on-surface px-5 py-3">{(b.adults ?? 0) + (b.children ?? 0)}</td>
              <td className="text-on-surface-variant px-5 py-3 capitalize">{b.source}</td>
              <td className="px-5 py-3">
                <div className="flex flex-wrap items-center gap-1.5">
                  <span
                    className={`inline-flex items-center rounded-full px-2.5 py-1 text-[11.5px] font-semibold ${STATUS_BADGE_CLASSES[b.status]}`}
                  >
                    {b.status}
                  </span>
                  {NEXT_STATUSES[b.status].map(({ status: next, label }) => {
                    const disabled = next === "approved" && conflictingApproval(b)
                    return (
                      <Button
                        key={next}
                        type="button"
                        variant="outline"
                        size="xs"
                        disabled={disabled}
                        title={
                          disabled
                            ? "These dates overlap an already-confirmed reservation."
                            : undefined
                        }
                        onClick={() => patchStatus({ id: b.id, data: { status: next } })}
                      >
                        {label}
                      </Button>
                    )
                  })}
                </div>
              </td>
              <td className="px-5 py-3">
                <button
                  type="button"
                  onClick={() => handleDelete(b.id)}
                  aria-label="Delete reservation"
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
