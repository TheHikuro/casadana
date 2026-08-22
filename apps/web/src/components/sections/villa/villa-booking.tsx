import {
  ApiError,
  useCreateBooking,
  useGetVillaAvailability,
  useGetVillaPricing,
  useGetVillaPricingSettings,
} from "@casa-dana/api"
import { keepPreviousData, useQueryClient } from "@tanstack/react-query"
import {
  addDays,
  addMonths,
  differenceInCalendarDays,
  endOfMonth,
  format,
  parseISO,
  startOfMonth,
} from "date-fns"
import { ArrowRight, ChevronLeft, ChevronRight, Minus, Plus } from "lucide-react"
import { useEffect, useMemo, useRef, useState } from "react"
import { Controller, useForm } from "react-hook-form"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import type { VillaData } from "@/constants/villas.const"
import { usePublishedRating } from "@/lib/use-published-rating"
import { cn } from "@/lib/utils"
import { m } from "@/paraglide/messages"
import { getLocale } from "@/paraglide/runtime"

interface VillaBookingProps {
  villaSlug: string
  booking: VillaData["booking"]
}

interface BookingFormValues {
  name: string
  checkIn: Date
  checkOut: Date
  guests: number
  email: string
  tel: string
  description: string
}

const getDaysOfWeek = () => [
  m.villa_booking_day_mo(),
  m.villa_booking_day_tu(),
  m.villa_booking_day_we(),
  m.villa_booking_day_th(),
  m.villa_booking_day_fr(),
  m.villa_booking_day_sa(),
  m.villa_booking_day_su(),
]

// Mirrors the API's own cap on a pricing window (maxCalendarDays in the
// pricing service). Kept in sync by hand — the API answers a wider window
// with a 422.
const MAX_WINDOW_DAYS = 400

function fmt(date: Date) {
  return date.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" })
}

function fmtMonth(date: Date) {
  return date.toLocaleDateString("en-US", { month: "long", year: "numeric" })
}

function nightsBetween(a: Date, b: Date) {
  return Math.max(0, Math.round((b.getTime() - a.getTime()) / 86_400_000))
}

function sameDay(a: Date, b: Date) {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  )
}

function startOfDay(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate())
}

// Default arrival = tomorrow (gives the guest at least one day of buffer);
// default departure = +7 nights from arrival.
function defaultDates(): { checkIn: Date; checkOut: Date } {
  const today = startOfDay(new Date())
  const checkIn = addDays(today, 1)
  const checkOut = addDays(checkIn, 7)
  return { checkIn, checkOut }
}

export default function VillaBooking({ villaSlug, booking }: VillaBookingProps) {
  // Computed once at mount — dates won't shift if the user keeps the page open
  // across midnight, which is fine for a booking flow.
  const defaults = useMemo(defaultDates, [])

  const { control, register, handleSubmit, watch, setValue, setError } = useForm<BookingFormValues>(
    {
      defaultValues: {
        name: "",
        checkIn: defaults.checkIn,
        checkOut: defaults.checkOut,
        guests: booking.defaultGuests,
        email: "",
        tel: "",
        description: "",
      },
    },
  )

  const checkIn = watch("checkIn")
  const checkOut = watch("checkOut")
  const guests = watch("guests")

  const [activeField, setActiveField] = useState<"in" | "out" | null>(null)
  const [viewMonth, setViewMonth] = useState<Date>(
    () => new Date(defaults.checkIn.getFullYear(), defaults.checkIn.getMonth(), 1),
  )
  const popRef = useRef<HTMLDivElement | null>(null)
  const ciRef = useRef<HTMLButtonElement | null>(null)
  const coRef = useRef<HTMLButtonElement | null>(null)

  const [submitted, setSubmitted] = useState(false)
  const [topLevelError, setTopLevelError] = useState<string | null>(null)
  const queryClient = useQueryClient()

  const { mutate: createBooking, isPending } = useCreateBooking({
    mutation: {
      onSuccess: () => {
        setSubmitted(true)
        setTopLevelError(null)
        queryClient.invalidateQueries({ queryKey: ["/api/villas", villaSlug, "availability"] })
      },
      onError: (err) => {
        if (err instanceof ApiError) {
          if (err.code === "VALIDATION") {
            const fieldMap: Record<string, keyof BookingFormValues> = {
              GuestName: "name",
              GuestEmail: "email",
              GuestPhone: "tel",
              CheckIn: "checkIn",
              CheckOut: "checkOut",
              Adults: "guests",
            }
            for (const part of err.message.split(";")) {
              const [rawField, tag] = part.split(":").map((s) => s.trim())
              const field = fieldMap[rawField ?? ""]
              if (field) setError(field, { type: tag || "invalid", message: tag })
            }
            setTopLevelError(null)
          } else if (err.code === "DATES_CONFLICT") {
            setTopLevelError(m.villa_booking_error_dates_conflict())
          } else if (err.code === "UNKNOWN_VILLA") {
            setTopLevelError(m.villa_booking_error_unknown_villa())
          } else {
            setTopLevelError(m.villa_booking_error_generic())
          }
        } else {
          setTopLevelError(m.villa_booking_error_generic())
        }
      },
    },
  })

  // Window covers both the visible calendar months AND the selected stay range
  // (so the displayed total stays accurate even when checkout is outside the
  // currently-viewed month).
  const queryWindow = useMemo(() => {
    const calFrom = startOfMonth(viewMonth)
    // The API window is [from, to) — exclusive — so pricing the last visible
    // day takes one extra day.
    const calTo = addDays(endOfMonth(addMonths(viewMonth, 1)), 1)
    let from = checkIn < calFrom ? checkIn : calFrom
    let to = checkOut > calTo ? checkOut : calTo
    // The API rejects a window wider than MAX_WINDOW_DAYS, which a guest can
    // reach by browsing a year away from their own dates. The stay wins that
    // budget — the sidebar total is the number that has to be right — and the
    // calendar keeps whatever is left.
    if (differenceInCalendarDays(to, from) > MAX_WINDOW_DAYS) {
      if (calFrom < checkIn) from = addDays(to, -MAX_WINDOW_DAYS)
      else to = addDays(from, MAX_WINDOW_DAYS)
    }
    return { from: format(from, "yyyy-MM-dd"), to: format(to, "yyyy-MM-dd") }
  }, [viewMonth, checkIn, checkOut])

  const { data: availability } = useGetVillaAvailability(
    villaSlug,
    { from: queryWindow.from, to: queryWindow.to },
    {
      query: {
        enabled: activeField !== null,
        placeholderData: keepPreviousData,
      },
    },
  )

  // Pricing is always enabled (not gated on calendar visibility) so the sidebar
  // total reflects the real back-office rates on first paint, before the user
  // opens the calendar.
  const { data: pricing } = useGetVillaPricing(
    villaSlug,
    { from: queryWindow.from, to: queryWindow.to },
    {
      query: {
        placeholderData: keepPreviousData,
      },
    },
  )

  // Base rate and both fees live in the back-office, per villa.
  // `GET .../pricing/settings` is public and reads back all-zero for a villa
  // that has never been configured, so an unset base rate falls back to the
  // hardcoded content value rather than showing €0, and an unset fee is simply
  // not charged.
  const { data: settings } = useGetVillaPricingSettings(villaSlug)
  const baseNightlyCents = settings?.base_price_cents || booking.nightly * 100
  const cleaningFeeCents = settings?.cleaning_fee_cents ?? 0
  const conciergeFeeCents = settings?.concierge_fee_cents ?? 0

  // Computed from approved reviews — the same figures the reviews section
  // further down the page shows. Null until a review is approved, in which case
  // the panel shows the rate alone rather than a rating no guest ever gave.
  const publishedRating = usePublishedRating(villaSlug)

  function datesToSet(ranges: Array<{ check_in: string; check_out: string }>): Set<string> {
    const set = new Set<string>()
    for (const r of ranges) {
      const start = parseISO(r.check_in)
      const end = parseISO(r.check_out)
      for (let d = start; d < end; d = addDays(d, 1)) {
        set.add(format(d, "yyyy-MM-dd"))
      }
    }
    return set
  }

  // Confirmed (approved/paid) nights are hard blocked. Pending nights are
  // provisionally held by an unconfirmed request — not guaranteed, but a new
  // booking on top of them would still conflict server-side, so they're shown
  // (distinctly) and are not selectable either.
  const blockedNights = useMemo(() => datesToSet(availability?.booked_ranges ?? []), [availability])
  const pendingNights = useMemo(
    () => datesToSet(availability?.pending_ranges ?? []),
    [availability],
  )

  const isBlocked = (date: Date) => blockedNights.has(format(date, "yyyy-MM-dd"))
  const isPendingDate = (date: Date) => pendingNights.has(format(date, "yyyy-MM-dd"))
  const isUnavailable = (date: Date) => isBlocked(date) || isPendingDate(date)

  // `nights` is the API's own answer for every day in the window: a per-date
  // override wins, then the highest matching season rule, then the villa's base
  // rate. Resolving it server-side is what keeps the panel in step with the
  // back-office — reading the raw `overrides` list alone missed every seasonal
  // rule the admin had configured.
  const nightPricesByDate = useMemo(() => {
    const map = new Map<string, number>()
    for (const night of pricing?.nights ?? []) {
      // A villa with no base rate configured resolves to 0; leaving it out
      // falls the night back to the editorial figure rather than advertising
      // a free night.
      if (night.price_cents > 0) map.set(night.date, night.price_cents)
    }
    return map
  }, [pricing])

  const priceCentsFor = (date: Date): number =>
    nightPricesByDate.get(format(date, "yyyy-MM-dd")) ?? baseNightlyCents

  const nights = nightsBetween(checkIn, checkOut)
  // Inline the per-night lookup to keep the dep array honest (priceCentsFor is
  // a plain function reference that React can't track).
  const nightsCents = useMemo(() => {
    let sum = 0
    for (let day = new Date(checkIn); day < checkOut; day = addDays(day, 1)) {
      const key = format(day, "yyyy-MM-dd")
      sum += nightPricesByDate.get(key) ?? baseNightlyCents
    }
    return sum
  }, [checkIn, checkOut, nightPricesByDate, baseNightlyCents])
  const nightsSubtotal = nightsCents / 100
  const cleaningFee = cleaningFeeCents / 100
  const conciergeFee = conciergeFeeCents / 100
  // Both back-office fees are charged once per stay, on top of the nights.
  const total = (nightsCents + cleaningFeeCents + conciergeFeeCents) / 100

  // Groups selected nights by their per-night price so the summary can show
  // "5 nights at €95" / "2 nights at €120" instead of a single blended total
  // whenever seasonal rates or overrides make the stay span more than one tier.
  const priceBreakdown = useMemo(() => {
    const counts = new Map<number, number>()
    for (let day = new Date(checkIn); day < checkOut; day = addDays(day, 1)) {
      const key = format(day, "yyyy-MM-dd")
      const priceCents = nightPricesByDate.get(key) ?? baseNightlyCents
      counts.set(priceCents, (counts.get(priceCents) ?? 0) + 1)
    }
    return Array.from(counts.entries())
      .map(([priceCents, count]) => ({ priceCents, count }))
      .sort((a, b) => a.priceCents - b.priceCents)
  }, [checkIn, checkOut, nightPricesByDate, baseNightlyCents])

  useEffect(() => {
    if (!activeField) return
    const handler = (e: MouseEvent) => {
      const target = e.target as Node
      if (
        popRef.current?.contains(target) ||
        ciRef.current?.contains(target) ||
        coRef.current?.contains(target)
      ) {
        return
      }
      setActiveField(null)
    }
    document.addEventListener("mousedown", handler)
    return () => document.removeEventListener("mousedown", handler)
  }, [activeField])

  const cells = useMemo(() => {
    const y = viewMonth.getFullYear()
    const mo = viewMonth.getMonth()
    const first = new Date(y, mo, 1)
    const startDow = (first.getDay() + 6) % 7
    const daysInMonth = new Date(y, mo + 1, 0).getDate()
    const prevDays = new Date(y, mo, 0).getDate()

    const out: Array<{ d: number; muted: boolean; date?: Date }> = []
    for (let i = startDow; i > 0; i--) {
      out.push({ d: prevDays - i + 1, muted: true })
    }
    for (let d = 1; d <= daysInMonth; d++) {
      out.push({ d, muted: false, date: new Date(y, mo, d) })
    }
    return out
  }, [viewMonth])

  const pickDate = (date: Date) => {
    if (isUnavailable(date)) return
    if (activeField === "in" || date < checkIn) {
      setValue("checkIn", date, { shouldDirty: true })
      if (checkOut <= date) {
        setValue("checkOut", new Date(date.getTime() + 7 * 86_400_000), { shouldDirty: true })
      }
      setActiveField("out")
    } else {
      setValue("checkOut", date, { shouldDirty: true })
      setActiveField(null)
    }
  }

  const openCal = (field: "in" | "out") => {
    setActiveField(field)
    const target = field === "in" ? checkIn : checkOut
    setViewMonth(new Date(target.getFullYear(), target.getMonth(), 1))
  }

  const onSubmit = (values: BookingFormValues) => {
    createBooking({
      data: {
        villa_slug: villaSlug,
        guest_name: values.name,
        guest_email: values.email,
        guest_phone: values.tel,
        check_in: format(values.checkIn, "yyyy-MM-dd"),
        check_out: format(values.checkOut, "yyyy-MM-dd"),
        adults: values.guests,
        children: 0,
        message: values.description,
        // The language this guest is reading the site in. It's persisted with
        // the booking so every email about it — including the approval sent
        // weeks later — is written in that language rather than in French.
        locale: getLocale(),
      },
    })
  }

  const inputClassName =
    "text-primary placeholder:text-on-surface-variant/50 mt-1 h-auto w-full rounded-none border-0 bg-transparent px-0 py-0 text-[15px] shadow-none focus-visible:border-0 focus-visible:ring-0 md:text-[15px]"

  if (submitted) {
    return (
      <aside
        id="book"
        className="bg-background border-outline-variant editorial-shadow border p-6 md:sticky md:top-28 md:p-8"
      >
        <div className="text-center">
          <h3 className="font-display text-primary text-[28px] italic">
            {m.villa_booking_success_title()}
          </h3>
          <p className="text-on-surface-variant mt-3 text-[15px]">
            {m.villa_booking_success_body()}
          </p>
        </div>
      </aside>
    )
  }

  return (
    <aside
      id="book"
      className="bg-background border-outline-variant editorial-shadow border p-6 md:sticky md:top-28 md:p-8"
    >
      {/* No headline nightly rate here: a single "from" figure misread the
          stay whenever a season rule or an override applied, and the real
          per-night prices are on the calendar and in the summary below. The
          rating carries the header on its own, and the whole strip drops when
          no review is approved rather than leaving an empty rule. */}
      {publishedRating && (
        <div className="border-outline-variant mb-6 flex flex-col gap-1 border-b pb-6 font-mono text-[11px] tracking-[0.1em]">
          <span className="text-secondary text-[13px] tracking-[2px]">
            {"★".repeat(publishedRating.filledStars)}
            {"☆".repeat(5 - publishedRating.filledStars)}
          </span>
          <span className="text-on-surface-variant">
            {publishedRating.score.toFixed(2)} ·{" "}
            {m.villa_booking_review_count({ count: publishedRating.count })}
          </span>
        </div>
      )}

      <form onSubmit={handleSubmit(onSubmit)} noValidate>
        <div className="border-outline-variant relative grid grid-cols-2 border">
          <button
            ref={ciRef}
            type="button"
            onClick={() => openCal("in")}
            className={cn(
              "border-outline-variant hover:bg-surface-container-low block border-r bg-white px-4 py-3.5 text-left transition-colors",
              activeField === "in" && "bg-surface-container-low",
            )}
          >
            <span className="text-on-surface-variant block font-mono text-[10px] tracking-[0.22em] uppercase">
              {m.villa_booking_check_in()}
            </span>
            <span className="font-display text-primary mt-1 block text-[17px] italic">
              {fmt(checkIn)}
            </span>
          </button>
          <button
            ref={coRef}
            type="button"
            onClick={() => openCal("out")}
            className={cn(
              "hover:bg-surface-container-low block bg-white px-4 py-3.5 text-left transition-colors",
              activeField === "out" && "bg-surface-container-low",
            )}
          >
            <span className="text-on-surface-variant block font-mono text-[10px] tracking-[0.22em] uppercase">
              {m.villa_booking_check_out()}
            </span>
            <span className="font-display text-primary mt-1 block text-[17px] italic">
              {fmt(checkOut)}
            </span>
          </button>

          {activeField !== null && (
            <div
              ref={popRef}
              className="border-outline-variant editorial-shadow absolute top-[calc(100%+8px)] right-0 left-0 z-30 border bg-white p-5"
            >
              <div className="mb-3 flex items-center justify-between">
                <button
                  type="button"
                  onClick={() =>
                    setViewMonth(new Date(viewMonth.getFullYear(), viewMonth.getMonth() - 1, 1))
                  }
                  aria-label={m.villa_booking_prev_month()}
                  className="border-outline-variant text-primary inline-flex h-7 w-7 items-center justify-center rounded-full border"
                >
                  <ChevronLeft size={14} />
                </button>
                <h4 className="font-display text-primary text-[17px] italic">
                  {fmtMonth(viewMonth)}
                </h4>
                <button
                  type="button"
                  onClick={() =>
                    setViewMonth(new Date(viewMonth.getFullYear(), viewMonth.getMonth() + 1, 1))
                  }
                  aria-label={m.villa_booking_next_month()}
                  className="border-outline-variant text-primary inline-flex h-7 w-7 items-center justify-center rounded-full border"
                >
                  <ChevronRight size={14} />
                </button>
              </div>
              <div className="grid grid-cols-7 gap-0.5">
                {getDaysOfWeek().map((d) => (
                  <div
                    key={d}
                    className="text-on-surface-variant py-2 text-center font-mono text-[9.5px] tracking-[0.15em] uppercase"
                  >
                    {d}
                  </div>
                ))}
                {cells.map((cell, i) => {
                  if (cell.muted || !cell.date) {
                    return (
                      <span
                        key={i}
                        className="text-outline-variant flex min-h-[42px] cursor-default items-center justify-center text-[13px]"
                      >
                        {cell.d}
                      </span>
                    )
                  }
                  if (isBlocked(cell.date)) {
                    return (
                      <span
                        key={i}
                        aria-disabled="true"
                        title={m.villa_booking_night_booked()}
                        className="text-on-surface-variant/40 flex min-h-[42px] cursor-not-allowed items-center justify-center text-[13px] line-through"
                      >
                        {cell.d}
                      </span>
                    )
                  }
                  if (isPendingDate(cell.date)) {
                    return (
                      <span
                        key={i}
                        aria-disabled="true"
                        title={m.villa_booking_night_pending()}
                        className="text-secondary decoration-secondary/60 flex min-h-[42px] cursor-not-allowed items-center justify-center text-[13px] underline decoration-dashed decoration-2 underline-offset-4"
                      >
                        {cell.d}
                      </span>
                    )
                  }
                  const isCI = sameDay(cell.date, checkIn)
                  const isCO = sameDay(cell.date, checkOut)
                  const inRange = cell.date > checkIn && cell.date < checkOut
                  const showPrice = !(isCI || isCO)
                  const priceCents = priceCentsFor(cell.date)
                  return (
                    <button
                      key={i}
                      type="button"
                      onClick={() => pickDate(cell.date as Date)}
                      className={cn(
                        "text-on-surface flex min-h-[42px] flex-col items-center justify-center gap-0.5 text-[13px] transition-colors",
                        inRange && "bg-secondary-container text-on-secondary-container",
                        (isCI || isCO) &&
                          "bg-primary text-on-primary mx-auto size-[42px] rounded-full",
                        !inRange && !isCI && !isCO && "hover:bg-surface-container-low",
                      )}
                    >
                      <span>{cell.d}</span>
                      {showPrice && (
                        <span className="text-on-surface-variant font-mono text-[9px] leading-none">
                          €{Math.round(priceCents / 100)}
                        </span>
                      )}
                    </button>
                  )
                })}
              </div>
            </div>
          )}
        </div>

        <Controller
          control={control}
          name="guests"
          render={({ field }) => (
            <div className="border-outline-variant flex items-center justify-between border border-t-0 bg-white px-4 py-3.5">
              <span className="text-on-surface-variant font-mono text-[10px] tracking-[0.22em] uppercase">
                {m.villa_booking_guests_label()}
              </span>
              <span className="inline-flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => field.onChange(Math.max(1, field.value - 1))}
                  aria-label={m.villa_booking_aria_remove_guest()}
                  className="border-outline-variant text-primary hover:bg-surface-container-low inline-flex h-7 w-7 items-center justify-center rounded-full border"
                >
                  <Minus size={12} />
                </button>
                <span className="font-display text-primary text-[17px] italic">
                  {guests}{" "}
                  {guests === 1 ? m.villa_booking_guest_singular() : m.villa_booking_guest_plural()}
                </span>
                <button
                  type="button"
                  onClick={() => field.onChange(Math.min(booking.maxGuests, field.value + 1))}
                  aria-label={m.villa_booking_aria_add_guest()}
                  className="border-outline-variant text-primary hover:bg-surface-container-low inline-flex h-7 w-7 items-center justify-center rounded-full border"
                >
                  <Plus size={12} />
                </button>
              </span>
            </div>
          )}
        />

        <div className="border-outline-variant mt-4 grid border border-b-0 bg-white">
          <label className="border-outline-variant block border-b px-4 py-3">
            <span className="text-on-surface-variant block font-mono text-[10px] tracking-[0.22em] uppercase">
              {m.villa_booking_name_label()}
            </span>
            <Input
              type="text"
              autoComplete="name"
              placeholder={m.villa_booking_name_placeholder()}
              className={inputClassName}
              {...register("name", { required: true })}
            />
          </label>
          <label className="border-outline-variant block border-b px-4 py-3">
            <span className="text-on-surface-variant block font-mono text-[10px] tracking-[0.22em] uppercase">
              {m.villa_booking_email_label()}
            </span>
            <Input
              type="email"
              inputMode="email"
              autoComplete="email"
              placeholder={m.villa_booking_email_placeholder()}
              className={inputClassName}
              {...register("email", { required: true })}
            />
          </label>
          <label className="border-outline-variant block border-b px-4 py-3">
            <span className="text-on-surface-variant block font-mono text-[10px] tracking-[0.22em] uppercase">
              {m.villa_booking_phone_label()}
            </span>
            <Input
              type="tel"
              inputMode="tel"
              autoComplete="tel"
              placeholder={m.villa_booking_phone_placeholder()}
              className={inputClassName}
              {...register("tel", { required: true })}
            />
          </label>
          <label className="border-outline-variant block border-b px-4 py-3">
            <span className="text-on-surface-variant block font-mono text-[10px] tracking-[0.22em] uppercase">
              {m.villa_booking_stay_about_label()}
            </span>
            <Textarea
              placeholder={m.villa_booking_stay_about_placeholder()}
              rows={3}
              className="text-primary placeholder:text-on-surface-variant/50 mt-1 min-h-0 w-full resize-none rounded-none border-0 bg-transparent px-0 py-0 text-[15px] leading-relaxed shadow-none focus-visible:border-0 focus-visible:ring-0 md:text-[15px]"
              {...register("description")}
            />
          </label>
        </div>

        {topLevelError && (
          <p className="border-error/30 bg-error-container/20 text-error mt-3 border px-3 py-2 text-[13px]">
            {topLevelError}
          </p>
        )}

        <Button
          type="submit"
          disabled={isPending}
          className="bg-primary text-on-primary hover:bg-primary-container mt-4 inline-flex h-auto w-full items-center justify-center gap-3 rounded-none px-6 py-[18px] font-mono text-[11px] tracking-[0.28em] uppercase disabled:opacity-60"
        >
          {isPending ? m.villa_booking_request_sending() : m.villa_booking_request_book()}
          {!isPending && <ArrowRight size={12} />}
        </Button>
        <Button
          type="button"
          variant="outline"
          className="text-primary border-outline-variant hover:bg-surface-container-low mt-2.5 h-auto w-full rounded-none border px-6 py-4 font-mono text-[11px] tracking-[0.28em] uppercase"
        >
          {m.villa_booking_contact_host()}
        </Button>
      </form>

      <div className="border-outline-variant mt-6 grid gap-3 border-t pt-5 text-[13.5px]">
        {priceBreakdown.length > 1 ? (
          priceBreakdown.map(({ priceCents, count }) => (
            <div key={priceCents} className="text-on-surface-variant flex justify-between">
              <span>
                {m.villa_booking_nights_at_price({
                  count,
                  nightsWord:
                    count === 1 ? m.villa_booking_night_singular() : m.villa_booking_night_plural(),
                  price: Math.round(priceCents / 100),
                })}
              </span>
              <span>€{((priceCents * count) / 100).toLocaleString()}</span>
            </div>
          ))
        ) : (
          <div className="text-on-surface-variant flex justify-between">
            <span>
              {nights}{" "}
              {nights === 1 ? m.villa_booking_night_singular() : m.villa_booking_night_plural()}
            </span>
            <span>€{nightsSubtotal.toLocaleString()}</span>
          </div>
        )}
        {cleaningFee > 0 && (
          <div className="text-on-surface-variant flex justify-between">
            <span>{m.villa_booking_cleaning_fee()}</span>
            <span>€{cleaningFee.toLocaleString()}</span>
          </div>
        )}
        {conciergeFee > 0 && (
          <div className="text-on-surface-variant flex justify-between">
            <span>{m.villa_booking_concierge_fee()}</span>
            <span>€{conciergeFee.toLocaleString()}</span>
          </div>
        )}
        <div className="font-display text-primary border-outline-variant mt-1 flex justify-between border-t pt-3.5 text-[22px] italic">
          <span>{m.villa_booking_total()}</span>
          <span>€{total.toLocaleString()}</span>
        </div>
      </div>
      <p className="text-on-surface-variant mt-4 text-center text-xs italic">
        {m.villa_booking_no_charge_note()}
      </p>
    </aside>
  )
}
