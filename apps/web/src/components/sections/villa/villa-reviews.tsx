import { type ReviewBreakdown, useGetVillaReviewMeta, useListVillaReviews } from "@casa-dana/api"

import type { VillaData } from "@/constants/villas.const"
import { m } from "@/paraglide/messages"

interface VillaReviewsProps {
  villaSlug: string
  data: VillaData["reviews"]
}

// The five bar labels in villas.const.ts are translated, generic wording
// ("Cleanliness", "Comfort", …) with no key of their own, so the pairing with
// the API's breakdown fields can only live here — positionally, in the order
// the labels are declared.
const BAR_CATEGORIES = [
  "cleanliness",
  "comfort",
  "location",
  "host",
  "value",
] as const satisfies ReadonlyArray<keyof ReviewBreakdown>

interface Bar {
  label: string
  pct: number
  value: string
}

interface Card {
  key: string
  initial: string
  name: string
  meta: string
  quote: string
  source: string
  stars: number
}

function Stars({ count }: { count: number }) {
  return (
    <span className="text-secondary text-[13px] tracking-[2px]">
      {"★".repeat(count)}
      {"☆".repeat(Math.max(0, 5 - count))}
    </span>
  )
}

export default function VillaReviews({ villaSlug, data }: VillaReviewsProps) {
  const { data: meta } = useGetVillaReviewMeta(villaSlug)
  const { data: list } = useListVillaReviews(villaSlug)

  // Real, admin-approved guest reviews — the only branch that shows actual
  // guest words. Everything under `data` is STATIC MARKETING SHOWCASE COPY from
  // villas.const.ts: invented sample reviews, NOT real guests. It renders while
  // the queries are in flight and for villas that have no approved review yet,
  // so the marketing page never shows an empty section. Real data wins as soon
  // as it exists.
  const published =
    meta && list && meta.display_count > 0 && list.reviews.length > 0
      ? { meta, reviews: list.reviews }
      : null

  const bars: Array<Bar> = published
    ? BAR_CATEGORIES.flatMap((category, i) => {
        const score = published.meta.breakdown[category]
        const label = data.bars[i]?.label
        // A null score means no approved review rated that category — drop the
        // bar rather than drawing it at zero.
        if (score === null || label === undefined) return []
        return [{ label, pct: (score / 5) * 100, value: score.toFixed(1) }]
      })
    : data.bars

  const cards: Array<Card> = published
    ? published.reviews.map((review) => ({
        key: review.id,
        initial: review.author_name.slice(0, 1).toUpperCase(),
        name: review.author_name,
        meta: review.meta,
        quote: review.body,
        source: review.source,
        stars: review.rating,
      }))
    : data.entries.map((entry) => ({ ...entry, key: entry.name }))

  const score = published ? published.meta.display_avg : data.score
  const count = published ? published.meta.display_count : data.count
  // The headline star row used to be five hardcoded glyphs; keep it truthful
  // once it stands for a computed average.
  const filledStars = Math.min(5, Math.max(0, Math.round(score)))

  return (
    <section id="reviews" className="bg-surface-container-low py-20 md:py-[140px]">
      <div className="mx-auto max-w-[1440px] px-6 md:px-10">
        <div className="mb-12 grid items-end gap-6 md:mb-16 md:grid-cols-[auto_1fr] md:gap-10">
          <div>
            <span className="text-secondary inline-flex items-center gap-3 font-mono text-[11px] tracking-[0.22em] uppercase before:block before:h-px before:w-6 before:bg-current">
              {data.chapter}
            </span>
            <h2 className="font-display text-primary mt-4 text-[clamp(40px,5.4vw,72px)] leading-none font-light tracking-[-0.025em]">
              {data.titleLead}
              <em className="italic-display block">{data.titleItalic}</em>
            </h2>
          </div>
          <p className="text-on-surface-variant max-w-[44ch] justify-self-start text-[15px] leading-relaxed md:justify-self-end md:text-right">
            {data.description}
          </p>
        </div>

        <div className="grid items-start gap-12 md:grid-cols-[1fr_1.6fr] md:gap-24">
          <aside className="md:sticky md:top-28">
            <div className="font-display text-primary text-[88px] leading-none font-light italic md:text-[120px]">
              {score.toFixed(2)}
              <small className="text-on-surface-variant ml-1 font-sans text-[16px] not-italic md:text-[22px]">
                /5
              </small>
            </div>
            <div className="text-secondary mt-3 mb-2 text-[18px] tracking-[3px] md:text-[22px] md:tracking-[4px]">
              {"★".repeat(filledStars)}
              {"☆".repeat(5 - filledStars)}
            </div>
            <div className="text-on-surface-variant mb-7 font-mono text-[11px] tracking-[0.18em] uppercase">
              {m.villa_reviews_based_on({ count })}
            </div>
            <div className="grid max-w-[320px] gap-3.5">
              {bars.map((b) => (
                <div
                  key={b.label}
                  className="text-on-surface-variant grid items-center gap-3 font-mono text-[10.5px] tracking-[0.12em] uppercase"
                  style={{ gridTemplateColumns: "100px 1fr 32px" }}
                >
                  <span>{b.label}</span>
                  <div className="bg-outline-variant relative h-[3px] overflow-hidden">
                    <span className="bg-secondary block h-full" style={{ width: `${b.pct}%` }} />
                  </div>
                  <span>{b.value}</span>
                </div>
              ))}
            </div>
          </aside>

          <div className="grid gap-8">
            {cards.map((rev) => (
              <article
                key={rev.key}
                className="bg-background border-outline-variant grid gap-5 border p-6 md:p-8"
              >
                <div className="flex flex-col items-start justify-between gap-3 md:flex-row md:items-center md:gap-4">
                  <div className="flex items-center gap-4">
                    <div className="bg-secondary-fixed text-on-secondary-container font-display inline-flex h-11 w-11 items-center justify-center rounded-full text-[18px] font-medium italic">
                      {rev.initial}
                    </div>
                    <div>
                      <div className="font-display text-primary text-[18px] italic">{rev.name}</div>
                      <div className="text-on-surface-variant mt-0.5 font-mono text-[10px] tracking-[0.15em] uppercase">
                        {rev.meta}
                      </div>
                    </div>
                  </div>
                  <Stars count={rev.stars} />
                </div>
                <p className="font-display text-on-surface relative text-[20px] leading-snug font-light italic md:text-[22px]">
                  <span className="text-secondary -mt-1 mr-1.5 inline-block align-top text-5xl leading-none">
                    “
                  </span>
                  {rev.quote}
                </p>
                <div className="text-on-surface-variant font-mono text-[10.5px] tracking-[0.18em] uppercase">
                  {rev.source}
                </div>
              </article>
            ))}
          </div>
        </div>
      </div>
    </section>
  )
}
