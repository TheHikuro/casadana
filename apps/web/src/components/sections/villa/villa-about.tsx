import { DynamicIcon, type IconName } from "lucide-react/dynamic"

import type { VillaData } from "@/constants/villas.const"

import VillaBooking from "./villa-booking"

interface VillaAboutProps {
  villaSlug: string
  about: VillaData["about"]
  booking: VillaData["booking"]
}

function FeatureCell({ feature }: { feature: { icon: IconName; label: string } }) {
  return (
    <div className="bg-background hover:bg-surface-container-low flex min-h-[120px] flex-col justify-between p-5 transition-colors md:min-h-[130px] md:p-6">
      <DynamicIcon name={feature.icon} size={26} strokeWidth={1.5} className="text-secondary" />
      <div className="font-mono text-[11px] tracking-[0.18em] uppercase">{feature.label}</div>
    </div>
  )
}

export default function VillaAbout({ villaSlug, about, booking }: VillaAboutProps) {
  return (
    <section id="about" className="mx-auto max-w-[1440px] px-6 py-20 md:px-10 md:py-[140px]">
      <div className="grid items-start gap-12 md:grid-cols-[1.15fr_0.85fr] md:gap-24">
        <div>
          <span className="text-secondary inline-flex items-center gap-3 font-mono text-[11px] tracking-[0.22em] uppercase before:block before:h-px before:w-6 before:bg-current">
            {about.chapter}
          </span>
          <h2 className="font-display text-primary mt-4 mb-7 text-[clamp(40px,5.4vw,72px)] leading-none font-light tracking-[-0.025em]">
            {about.titleLead}
            <br />
            <em className="italic-display text-secondary">{about.titleItalic}</em>
          </h2>
          <p className="font-display text-on-surface mb-8 max-w-[28ch] text-2xl leading-tight font-light tracking-[-0.01em]">
            {about.lead}
          </p>
          <div className="text-on-surface-variant mb-12 grid max-w-[60ch] gap-4 text-[15.5px] leading-relaxed">
            {about.body.map((paragraph) => (
              <p key={paragraph}>{paragraph}</p>
            ))}
          </div>

          <div className="border-outline-variant bg-outline-variant grid grid-cols-2 gap-px border md:grid-cols-4">
            {about.features.map((feature) => (
              <FeatureCell key={feature.label} feature={feature} />
            ))}
          </div>

          <div className="mt-6 grid grid-cols-1 gap-6 md:grid-cols-3 md:gap-8">
            {about.meta.map((m) => (
              <div key={m.label} className="border-outline-variant border-t pt-4">
                <div className="text-on-surface-variant font-mono text-[10.5px] tracking-[0.22em] uppercase">
                  {m.label}
                </div>
                <div className="font-display text-primary mt-1.5 text-[22px] italic">{m.value}</div>
              </div>
            ))}
          </div>
        </div>

        <VillaBooking villaSlug={villaSlug} booking={booking} />
      </div>
    </section>
  )
}
