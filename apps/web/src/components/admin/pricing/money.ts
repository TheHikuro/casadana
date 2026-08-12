// The API stores every amount in cents; every money field on this screen is in
// euros. Convert here and nowhere else.

export function centsToEuroInput(cents: number): string {
  return String(cents / 100)
}

/** A blank or unparseable field reads as 0 rather than submitting NaN. */
export function euroInputToCents(input: string): number {
  const euros = Number.parseFloat(input)
  return Number.isFinite(euros) ? Math.round(euros * 100) : 0
}
