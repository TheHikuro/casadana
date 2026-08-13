import { m } from "@/paraglide/messages"

export default function MarqueeRibbon() {
  const phrases = [
    m.home_marquee_hosted_since(),
    m.home_marquee_direct_booking(),
    m.home_marquee_two_casas(),
    m.home_marquee_region(),
  ]

  return (
    <div className="bg-primary text-on-primary border-primary-container overflow-hidden border-t py-6 md:py-7">
      <div
        className="flex w-max items-center gap-16 whitespace-nowrap"
        style={{ animation: "marquee 60s linear infinite" }}
      >
        {[...phrases, ...phrases].map((phrase, index) => (
          <span
            key={`${phrase}-${index}`}
            className="font-display inline-flex items-center gap-16 text-xl font-light italic md:text-[26px]"
          >
            {phrase}
            <span className="text-secondary-fixed text-sm opacity-75">✦</span>
          </span>
        ))}
      </div>
    </div>
  )
}
