import { getVilla } from "@/constants/villas.const"

export type Property = "casadana" | "casacasay"

// A villa that was never configured reads back all-zero from the API. Offering
// those zeros as the form's starting point invites saving a €0 nightly rate —
// and `min_nights: 0`, which the PUT schema rejects outright — so an unset
// field opens on a sensible default instead.

/**
 * The nightly rate the public page already advertises for this villa
 * (`booking.nightly` in villas.const.ts), in cents. Read from the content
 * constants rather than duplicated here so the two never drift.
 */
export function defaultBasePriceCents(property: Property): number {
  return (getVilla(property)?.booking.nightly ?? 0) * 100
}

/** Matches both the DB column default and the PUT schema's minimum. */
export const DEFAULT_MIN_NIGHTS = 1
