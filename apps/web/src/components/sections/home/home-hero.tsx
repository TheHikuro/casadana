import bgHome from "@/assets/shared/bg-home.jpeg"
import { type PublishedRating, usePublishedRating } from "@/lib/use-published-rating"
import { m } from "@/paraglide/messages"
import { getLocale } from "@/paraglide/runtime"

interface VitalStat {
  label: string
  value: string
}

/**
 * The rating for the collection as a whole: each villa's published score,
 * weighted by how many approved reviews stand behind it, so the home page can
 * never quote a figure the two villa pages don't add up to.
 *
 * A villa with no approved review contributes nothing; with none approved
 * anywhere the hero shows a plain 0 rather than an invented score.
 */
function useCollectionRatingValue(): string {
  const rated = [usePublishedRating("casadana"), usePublishedRating("casacasay")].filter(
    (rating): rating is PublishedRating => rating !== null,
  )
  const reviewCount = rated.reduce((total, rating) => total + rating.count, 0)
  const score =
    reviewCount === 0
      ? 0
      : rated.reduce((total, rating) => total + rating.score * rating.count, 0) / reviewCount

  return m.home_hero_stat_rating_value({
    score: new Intl.NumberFormat(getLocale(), { maximumFractionDigits: 1 }).format(score),
  })
}

function HeroStats({ stats }: { stats: VitalStat[] }) {
  return (
    <div className="grid grid-cols-2 border-y border-white/25 py-6 md:grid-cols-4 md:py-7">
      {stats.map((stat, index) => (
        <div
          key={stat.label}
          className={`px-4 md:px-7 ${index !== 0 ? "md:border-l md:border-white/20" : ""} ${
            index < stats.length - 2 ? "border-b border-white/20 pb-3 md:border-b-0 md:pb-0" : ""
          }`}
        >
          <div className="font-mono text-[10px] tracking-[0.22em] text-white/70 uppercase">
            {stat.label}
          </div>
          <div className="font-display mt-1.5 text-xl italic md:text-[22px]">{stat.value}</div>
        </div>
      ))}
    </div>
  )
}

export default function HomeHero() {
  const ratingValue = useCollectionRatingValue()

  const stats: VitalStat[] = [
    { label: m.home_hero_stat_properties_label(), value: m.home_hero_stat_properties_value() },
    { label: m.home_hero_stat_sleeps_label(), value: m.home_hero_stat_sleeps_value() },
    { label: m.home_hero_stat_region_label(), value: m.home_hero_stat_region_value() },
    { label: m.home_hero_stat_rating_label(), value: ratingValue },
  ]

  return (
    <section className="relative h-screen min-h-[720px] overflow-hidden text-white">
      <div className="absolute inset-0">
        <img
          src={bgHome}
          alt={m.home_hero_image_alt()}
          className="animate-float-bg h-full w-full object-cover"
          style={{ animation: "float-bg 22s ease-in-out infinite alternate" }}
        />
        <div
          className="absolute inset-0"
          style={{
            background:
              "linear-gradient(180deg, oklch(23.6% 0.108 253 / 0.5) 0%, oklch(23.6% 0.108 253 / 0.1) 35%, oklch(23.6% 0.108 253 / 0.65) 100%)",
          }}
        />
      </div>

      <div className="relative z-10 mx-auto grid h-full max-w-[1440px] grid-rows-[1fr_auto_auto] items-end px-6 pt-28 md:px-10">
        <div>
          <div className="mb-7 flex flex-wrap gap-x-4 gap-y-2">
            <span className="font-mono text-[11px] tracking-[0.22em] text-white/85 uppercase">
              {m.home_hero_meta_collection()}
            </span>
            <span className="font-mono text-[11px] tracking-[0.22em] text-white/85 uppercase">
              {m.home_hero_meta_region()}
            </span>
            <span className="font-mono text-[11px] tracking-[0.22em] text-white/85 uppercase">
              {m.home_hero_meta_host()}
            </span>
          </div>
          <h1
            className="font-display mb-8 text-[clamp(64px,9vw,156px)] leading-[0.88] font-light tracking-[-0.035em] md:mb-14"
            style={{
              fontVariationSettings: '"SOFT" 40',
              textShadow: "0 2px 24px oklch(23.6% 0.108 253 / 0.35)",
            }}
          >
            {m.home_hero_title_prefix()}{" "}
            <em className="italic-display">{m.home_hero_title_dana()}</em>
            <br />
            {m.home_hero_title_and()}{" "}
            <em className="italic-display">{m.home_hero_title_casay()}</em>
          </h1>
        </div>

        <HeroStats stats={stats} />

        <div className="flex flex-col items-start justify-between gap-4 py-8 md:flex-row md:items-center md:py-9">
          <p className="max-w-[46ch] text-sm leading-relaxed text-white/85 md:text-[14.5px]">
            {m
              .home_hero_paragraph()
              .split("\n")
              .map((line) => (
                <span key={line} className="block">
                  {line}
                </span>
              ))}
          </p>
          <a
            href="#collection"
            className="inline-flex items-center gap-3 font-mono text-[11px] tracking-[0.22em] text-white uppercase"
          >
            <span>{m.home_hero_scroll_cue_label()}</span>
            <span className="relative h-px w-16 overflow-hidden bg-white/40">
              <span
                className="absolute inset-0 origin-left bg-white"
                style={{ animation: "scroll-cue 2.4s ease-in-out infinite" }}
              />
            </span>
            <span>{m.home_hero_scroll_cue_below()}</span>
          </a>
        </div>
      </div>
    </section>
  )
}
