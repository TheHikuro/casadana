import type { VillaData } from "@/constants/villas.const"
import { m } from "@/paraglide/messages"

interface VillaReviewsProps {
  data: VillaData["reviews"]
}

function Stars({ count }: { count: number }) {
  return (
    <span className="text-secondary text-[13px] tracking-[2px]">
      {"★".repeat(count)}
      {"☆".repeat(Math.max(0, 5 - count))}
    </span>
  )
}

export default function VillaReviews({ data }: VillaReviewsProps) {
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
              {data.score.toFixed(2)}
              <small className="text-on-surface-variant ml-1 font-sans text-[16px] not-italic md:text-[22px]">
                /5
              </small>
            </div>
            <div className="text-secondary mt-3 mb-2 text-[18px] tracking-[3px] md:text-[22px] md:tracking-[4px]">
              ★★★★★
            </div>
            <div className="text-on-surface-variant mb-7 font-mono text-[11px] tracking-[0.18em] uppercase">
              {m.villa_reviews_based_on({ count: data.count })}
            </div>
            <div className="grid max-w-[320px] gap-3.5">
              {data.bars.map((b) => (
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
            {data.entries.map((rev) => (
              <article
                key={rev.name}
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
