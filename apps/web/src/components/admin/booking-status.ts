import type { BookingSource, BookingStatus } from "@casa-dana/api"

// The API enums stay on the wire in English (pending / approved / …); these
// maps are display-only, for the French dashboard.
export const BOOKING_STATUS_LABELS: Record<BookingStatus, string> = {
  pending: "En attente",
  approved: "Approuvée",
  paid: "Payée",
  rejected: "Refusée",
  cancelled: "Annulée",
}

export const BOOKING_SOURCE_LABELS: Record<BookingSource, string> = {
  direct: "Direct",
  airbnb: "Airbnb",
  booking_com: "Booking.com",
}
