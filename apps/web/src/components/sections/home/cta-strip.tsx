import { Link } from "@tanstack/react-router"
import { ArrowRight } from "lucide-react"

import { m } from "@/paraglide/messages"

export default function CtaStrip() {
  return (
    <section className="bg-secondary-fixed text-on-secondary-container py-20 md:py-24">
      <div className="mx-auto grid max-w-[1440px] items-center gap-8 px-6 md:grid-cols-[1fr_auto] md:gap-16 md:px-10">
        <h2 className="font-display m-0 text-[clamp(36px,4.6vw,56px)] leading-tight font-light tracking-[-0.02em]">
          {m.home_cta_strip_lead()} <em className="italic-display">{m.home_cta_strip_em()}</em>{" "}
          {m.home_cta_strip_tail()}
        </h2>
        <Link
          to="/villa/$villaId"
          params={{ villaId: "casadana" }}
          className="bg-primary text-on-primary hover:bg-primary-container inline-flex items-center justify-center gap-3 px-7 py-[18px] font-mono text-[11px] tracking-[0.28em] uppercase transition-colors"
        >
          {m.home_cta_strip_button()}
          <ArrowRight size={14} />
        </Link>
      </div>
    </section>
  )
}
