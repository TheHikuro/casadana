import { ArrowRight, Plus } from "lucide-react"

import type { VillaData } from "@/constants/villas.const"
import { m } from "@/paraglide/messages"

interface VillaFaqProps {
  data: VillaData["faq"]
}

export default function VillaFaq({ data }: VillaFaqProps) {
  if (!data) return null
  return (
    <section className="mx-auto max-w-[1440px] px-6 py-20 md:px-10 md:py-[140px]">
      <div className="grid items-start gap-12 md:grid-cols-[1fr_1.4fr] md:gap-24">
        <div>
          <span className="text-secondary inline-flex items-center gap-3 font-mono text-[11px] tracking-[0.22em] uppercase before:block before:h-px before:w-6 before:bg-current">
            {data.chapter}
          </span>
          <h2 className="font-display text-primary mt-4 mb-6 text-[clamp(40px,5.4vw,72px)] leading-none font-light tracking-[-0.025em]">
            {data.titleLead}
            <em className="italic-display block">{data.titleItalic}</em>
          </h2>
          <p className="text-on-surface-variant mb-8 max-w-[36ch] leading-relaxed">{data.intro}</p>
          <a
            href="mailto:hello@casa-dana.com"
            className="text-primary border-outline-variant hover:bg-surface-container-low inline-flex items-center gap-3 border px-6 py-4 font-mono text-[11px] tracking-[0.28em] uppercase transition-colors"
          >
            {m.villa_faq_email_host()}
            <ArrowRight size={12} />
          </a>
        </div>

        <div>
          {data.entries.map((entry, i) => (
            <details
              key={entry.question}
              className="border-outline-variant group border-t py-5 last:border-b"
              open={i === 0}
            >
              <summary className="flex cursor-pointer list-none items-baseline justify-between gap-6">
                <span className="font-display text-primary text-[20px] font-normal italic md:text-[22px]">
                  {entry.question}
                </span>
                <span className="border-outline-variant text-primary group-open:bg-primary group-open:text-on-primary group-open:border-primary inline-flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-full border transition-all duration-300 group-open:rotate-45">
                  <Plus size={14} />
                </span>
              </summary>
              <p className="text-on-surface-variant max-w-[60ch] pt-4 text-[14.5px] leading-relaxed">
                {entry.answer}
              </p>
            </details>
          ))}
        </div>
      </div>
    </section>
  )
}
