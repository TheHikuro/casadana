import { Link } from "@tanstack/react-router"
import { ArrowRight } from "lucide-react"

import type { VillaData } from "@/constants/villas.const"

interface VillaSisterProps {
  data: VillaData["sister"]
}

export default function VillaSister({ data }: VillaSisterProps) {
  return (
    <section className="bg-primary py-20 text-white md:py-[140px]">
      <div className="mx-auto grid max-w-[1440px] items-center gap-12 px-6 md:grid-cols-2 md:gap-20 md:px-10">
        <div>
          <span className="text-secondary-fixed inline-flex items-center gap-3 font-mono text-[11px] tracking-[0.22em] uppercase before:block before:h-px before:w-6 before:bg-current">
            {data.chapter}
          </span>
          <h2 className="font-display mt-4 mb-6 text-[clamp(36px,5vw,60px)] leading-[1.04] font-light tracking-[-0.025em]">
            {data.titleLead}{" "}
            <em className="italic-display text-secondary-fixed">{data.titleItalic}</em>
          </h2>
          <p className="mb-8 max-w-[48ch] leading-relaxed opacity-80">{data.description}</p>
          <Link
            to="/villa/$villaId"
            params={{ villaId: data.targetId }}
            className="bg-secondary-fixed text-on-secondary-container hover:bg-background inline-flex items-center gap-3 px-8 py-5 font-mono text-[11px] tracking-[0.28em] uppercase transition-colors"
          >
            {data.button}
            <ArrowRight size={12} />
          </Link>
        </div>
        <div className="aspect-[4/5] overflow-hidden">
          <img
            src={data.image}
            alt={data.titleItalic}
            className="h-full w-full object-cover transition-transform duration-700 hover:scale-[1.04]"
          />
        </div>
      </div>
    </section>
  )
}
