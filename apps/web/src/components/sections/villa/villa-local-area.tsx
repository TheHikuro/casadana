import type { VillaData } from "@/constants/villas.const"
import { m } from "@/paraglide/messages"

interface VillaLocalAreaProps {
  data: VillaData["localArea"]
}

export default function VillaLocalArea({ data }: VillaLocalAreaProps) {
  return (
    <section id="location" className="mx-auto max-w-[1440px] px-6 py-20 md:px-10 md:py-[140px]">
      <div className="grid items-center gap-12 md:grid-cols-2 md:gap-24">
        <div className="relative pt-6 pr-12 pb-12 md:pr-16 md:pb-16">
          <div className="aspect-[4/5] overflow-hidden">
            <img src={data.mainImage} alt={data.titleLead} className="h-full w-full object-cover" />
          </div>
          {data.overlapImage && (
            <div className="border-background editorial-shadow absolute right-0 bottom-0 hidden h-48 w-48 overflow-hidden border-[10px] md:block md:h-60 md:w-60">
              <img
                src={data.overlapImage}
                alt={m.villa_local_area_overlap_alt()}
                className="h-full w-full object-cover"
              />
            </div>
          )}
          <div
            className="bg-secondary-fixed text-on-secondary-container editorial-shadow absolute top-0 left-0 flex h-28 w-28 flex-col items-center justify-center rounded-full text-center md:h-32 md:w-32"
            style={{ transform: "rotate(-8deg)" }}
          >
            <span className="font-display text-3xl leading-none italic md:text-[34px]">
              {data.stampBig}
            </span>
            <span className="mt-1 max-w-[7ch] font-mono text-[8.5px] leading-tight tracking-[0.22em] uppercase">
              {data.stampSmall.split("\n").map((line, i) => (
                <span key={i} className="block">
                  {line}
                </span>
              ))}
            </span>
          </div>
        </div>

        <div>
          <span className="text-secondary inline-flex items-center gap-3 font-mono text-[11px] tracking-[0.22em] uppercase before:block before:h-px before:w-6 before:bg-current">
            {data.chapter}
          </span>
          <h2 className="font-display text-primary mt-4 mb-7 text-[clamp(40px,5.4vw,72px)] leading-none font-light tracking-[-0.025em]">
            {data.titleLead}
            <em className="italic-display text-secondary block">{data.titleItalic}</em>
          </h2>
          <p className="text-on-surface-variant mb-9 max-w-[50ch] text-[15.5px] leading-relaxed">
            {data.description}
          </p>

          <div>
            {data.points.map((p, i) => (
              <div
                key={p.number}
                className={`border-outline-variant grid grid-cols-1 gap-1.5 border-t py-5 md:grid-cols-[60px_1fr] md:gap-5 ${
                  i === data.points.length - 1 ? "border-b" : ""
                }`}
              >
                <div className="text-secondary pt-1 font-mono text-[11px] tracking-[0.2em]">
                  {p.number}
                </div>
                <div>
                  <h4 className="font-display text-primary mb-1.5 text-[22px] font-normal italic">
                    {p.title}
                  </h4>
                  <p className="text-on-surface-variant text-[14.5px] leading-relaxed">
                    {p.description}
                  </p>
                  <span className="text-on-surface-variant mt-2 inline-block font-mono text-[10.5px] tracking-[0.18em] uppercase">
                    {p.distance}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  )
}
