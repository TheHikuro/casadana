import type { ExperienceEntry, VillaData } from "@/constants/villas.const"

interface VillaExperiencesProps {
  data: VillaData["experiences"]
}

function ExperienceIcon({ icon }: { icon: ExperienceEntry["icon"] }) {
  switch (icon) {
    case "kart":
      return (
        <svg
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.5"
        >
          <circle cx="6" cy="17" r="3" />
          <circle cx="18" cy="17" r="3" />
          <path d="M9 17h6M2 14h20l-3-7H7l-3 4-2 3z" />
        </svg>
      )
    case "wind":
      return (
        <svg
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.5"
        >
          <path d="M2 21c2 0 2-1 4-1s2 1 4 1 2-1 4-1 2 1 4 1 2-1 4-1M12 4v12M5 16l7-12 7 12" />
        </svg>
      )
    case "splash":
      return (
        <svg
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.5"
        >
          <path d="M2 16c2 0 2-1 4-1s2 1 4 1 2-1 4-1 2 1 4 1 2-1 4-1M6 12V4l6 4V4l6 4M9 12h.01M15 12h.01" />
        </svg>
      )
    case "kayak":
      return (
        <svg
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.5"
        >
          <path d="M3 17l9-5 9 5M3 17l4-12 5 5 4-4 5 11" />
        </svg>
      )
    case "fairway":
      return (
        <svg
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.5"
        >
          <path d="M5 21V7l8-3v17M13 11h6M13 15h6" />
        </svg>
      )
    case "olive":
      return (
        <svg
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.5"
        >
          <path d="M12 22c5 0 9-4 9-9 0-1.5-.5-3-1.5-4.5C18 7 16 6 12 6S6 7 4.5 8.5C3.5 10 3 11.5 3 13c0 5 4 9 9 9zM12 6V2" />
        </svg>
      )
    case "salt":
      return (
        <svg
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.5"
        >
          <path d="M3 20c4 0 4-4 9-4s5 4 9 4M3 14c4 0 4-4 9-4s5 4 9 4" />
        </svg>
      )
    case "spa":
      return (
        <svg
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.5"
        >
          <path d="M12 22s-8-6-8-13a8 8 0 1 1 16 0c0 7-8 13-8 13z" />
          <circle cx="12" cy="9" r="3" />
        </svg>
      )
  }
}

export default function VillaExperiences({ data }: VillaExperiencesProps) {
  if (!data) return null
  return (
    <section id="experiences" className="bg-primary overflow-hidden py-20 text-white md:py-[140px]">
      <div className="mx-auto max-w-[1440px] px-6 md:px-10">
        <div className="mb-12 grid items-end gap-6 md:mb-16 md:grid-cols-[auto_1fr] md:gap-10">
          <div>
            <span className="text-secondary-fixed inline-flex items-center gap-3 font-mono text-[11px] tracking-[0.22em] uppercase before:block before:h-px before:w-6 before:bg-current">
              {data.chapter}
            </span>
            <h2 className="font-display mt-4 text-[clamp(40px,5.4vw,72px)] leading-none font-light tracking-[-0.025em] text-white">
              {data.titleLead}
              <em className="italic-display text-secondary-fixed block">{data.titleItalic}</em>
            </h2>
          </div>
          <p className="max-w-[44ch] justify-self-start text-[15px] leading-relaxed text-white/70 md:justify-self-end md:text-right">
            {data.description}
          </p>
        </div>

        <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
          {data.entries.map((entry) => (
            <article
              key={entry.number}
              className="relative flex min-h-[320px] flex-col justify-between overflow-hidden border p-7"
              style={{
                background: "oklch(28% 0.085 256)",
                borderColor: "oklch(36% 0.07 257)",
              }}
            >
              <span className="absolute top-6 right-6 font-mono text-[10px] tracking-[0.15em] uppercase opacity-50">
                {entry.tag}
              </span>
              <div>
                <div className="text-secondary-fixed font-mono text-[10.5px] tracking-[0.22em]">
                  {entry.number}
                </div>
                <div className="text-secondary-fixed my-5 flex h-[52px] w-[52px] items-center justify-center rounded-full border border-white/30">
                  <ExperienceIcon icon={entry.icon} />
                </div>
                <h3 className="font-display mb-2.5 text-[28px] leading-tight font-normal tracking-[-0.01em] italic">
                  {entry.title}
                </h3>
                <p className="text-[13.5px] leading-relaxed text-white/75">{entry.description}</p>
              </div>
              <div className="text-secondary-fixed mt-6 font-mono text-[10.5px] tracking-[0.22em]">
                {entry.location}
              </div>
            </article>
          ))}
        </div>
      </div>
    </section>
  )
}
