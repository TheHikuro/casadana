import { useQueryClient } from "@tanstack/react-query"
import { Trash2 } from "lucide-react"

import {
  type BookingResponse,
  type BookingStatus,
  type PatchBookingRequestStatus,
  getListBookingsQueryKey,
  useDeleteBooking,
  usePatchBooking,
} from "@casa-dana/api"
import { Button } from "@/components/ui/button"
import { useToast } from "@/components/ui/toast"

const NEXT_STATUSES: Record<BookingStatus, Array<{ status: PatchBookingRequestStatus; label: string }>> = {
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
  status?: BookingStatus
  page: number
}

export default function ReservationTable({ bookings, property, status, page }: ReservationTableProps) {
  const queryClient = useQueryClient()
  const { toast } = useToast()
  // Invalidate at the endpoint prefix (not the exact page/limit key) so this
  // also refreshes the stat row's separate limit:1 queries, not just this page.
  const bookingsQueryKey = getListBookingsQueryKey()

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
      <div className="px-5 py-10 text-center text-[13.5px] text-on-surface-variant">
        No reservations yet — new "Request to Book" submissions from the public site will land here.
      </div>
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[720px] border-collapse text-[13px]">
        <thead>
          <tr className="border-b border-outline-variant bg-surface-container-low text-left text-[10.5px] font-semibold tracking-[0.08em] text-on-surface-variant uppercase">
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
            <tr key={b.id} className="border-b border-outline-variant last:border-0">
              <td className="px-5 py-3">
                <p className="font-semibold text-on-surface">{b.guest_name}</p>
                <p className="mt-0.5 font-mono text-[11px] text-on-surface-variant">{b.guest_email}</p>
              </td>
              <td className="px-5 py-3 font-mono text-on-surface">
                {b.check_in} → {b.check_out}
              </td>
              <td className="px-5 py-3 text-on-surface">{(b.adults ?? 0) + (b.children ?? 0)}</td>
              <td className="px-5 py-3 text-on-surface-variant capitalize">{b.source}</td>
              <td className="px-5 py-3">
                <div className="flex flex-wrap items-center gap-1.5">
                  <span
                    className={`inline-flex items-center rounded-full px-2.5 py-1 text-[11.5px] font-semibold ${STATUS_BADGE_CLASSES[b.status]}`}
                  >
                    {b.status}
                  </span>
                  {NEXT_STATUSES[b.status].map(({ status: next, label }) => (
                    <Button
                      key={next}
                      type="button"
                      variant="outline"
                      size="xs"
                      onClick={() => patchStatus({ id: b.id, data: { status: next } })}
                    >
                      {label}
                    </Button>
                  ))}
                </div>
              </td>
              <td className="px-5 py-3">
                <button
                  type="button"
                  onClick={() => handleDelete(b.id)}
                  aria-label="Delete reservation"
                  className="rounded-md p-1.5 text-on-surface-variant hover:bg-error-container hover:text-on-error-container"
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
