import { m } from "@/paraglide/messages"

interface Stat {
  label: string
  value: string
}

interface VillaHeroProps {
  image: string
  eyebrow: Array<string>
  titlePrefix: string
  titleItalic: string
  titleSuffix: string
  stats: Array<Stat>
  price: number
  priceLabel: string
}

export default function VillaHero({
  image,
  eyebrow,
  titlePrefix,
  titleItalic,
  titleSuffix,
  stats,
  price,
  priceLabel,
}: VillaHeroProps) {
  return (
    <section className="relative h-screen min-h-[600px] overflow-hidden text-white md:min-h-[720px]">
      <div className="absolute inset-0">
        <img
          src={image}
          alt={`${titlePrefix}${titleItalic}`}
          className="h-full w-full object-cover"
          style={{ animation: "float-bg 22s ease-in-out infinite alternate" }}
        />
        <div
          className="absolute inset-0"
          style={{
            background:
              "linear-gradient(180deg, oklch(23.6% 0.108 253 / 0.45) 0%, oklch(23.6% 0.108 253 / 0.05) 35%, oklch(23.6% 0.108 253 / 0.6) 100%)",
          }}
        />
      </div>

      <div className="relative z-10 mx-auto grid h-full max-w-[1440px] grid-rows-[1fr_auto_auto] items-end px-6 pt-24 md:px-10 md:pt-32">
        <div>
          <div className="mb-7 flex flex-wrap items-baseline gap-x-4 gap-y-2">
            {eyebrow.map((line) => (
              <span
                key={line}
                className="font-mono text-[11px] tracking-[0.22em] text-white/85 uppercase"
              >
                {line}
              </span>
            ))}
          </div>
          <h1
            className="font-display mb-8 text-[clamp(64px,9vw,156px)] leading-[0.88] font-light tracking-[-0.035em] md:mb-14"
            style={{
              fontVariationSettings: '"SOFT" 40',
              textShadow: "0 2px 24px oklch(23.6% 0.108 253 / 0.35)",
            }}
          >
            {titlePrefix}
            <em className="italic-display">{titleItalic}</em>
            <br />
            <span className="text-[0.6em] md:text-[0.55em]">{titleSuffix}</span>
          </h1>
        </div>

        <div className="grid grid-cols-2 border-y border-white/25 py-5 md:grid-cols-5 md:py-6">
          {stats.map((stat, i) => (
            <div
              key={stat.label}
              className={`px-4 md:px-7 ${i !== 0 ? "md:border-l md:border-white/20" : ""} ${
                i < stats.length - (stats.length % 2 === 0 ? 2 : 1)
                  ? "border-b border-white/20 pb-3 md:border-b-0 md:pb-0"
                  : ""
              }`}
            >
              <div className="font-mono text-[10px] tracking-[0.22em] text-white/70 uppercase">
                {stat.label}
              </div>
              <div className="font-display mt-1.5 text-xl italic md:text-[22px]">{stat.value}</div>
            </div>
          ))}
        </div>

        <div className="flex flex-col items-start justify-between gap-4 py-7 md:flex-row md:items-center md:py-8">
          <div className="font-display text-[26px] italic md:text-[28px]">
            €{price}
            <small className="ml-1 font-sans text-[12px] tracking-[0.05em] text-white/75 not-italic">
              {priceLabel}
            </small>
          </div>
          <a
            href="#about"
            className="inline-flex items-center gap-3 font-mono text-[11px] tracking-[0.22em] uppercase"
          >
            <span>{m.villa_hero_scroll()}</span>
            <span className="relative h-px w-16 overflow-hidden bg-white/40">
              <span
                className="absolute inset-0 origin-left bg-white"
                style={{ animation: "scroll-cue 2.4s ease-in-out infinite" }}
              />
            </span>
            <span>{m.villa_hero_discover()}</span>
          </a>
        </div>
      </div>
    </section>
  )
}
