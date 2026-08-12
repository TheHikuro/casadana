import { m } from "@/paraglide/messages"

export default function IntroCollection() {
  return (
    <section
      id="collection"
      className="mx-auto max-w-[1440px] px-6 pt-32 pb-16 md:px-10 md:pt-36 md:pb-20"
    >
      <div className="grid items-end gap-8 md:grid-cols-[auto_1fr] md:gap-24">
        <div>
          <span className="text-secondary inline-flex items-center gap-3 font-mono text-[11px] tracking-[0.22em] uppercase before:block before:h-px before:w-6 before:bg-current">
            {m.home_intro_index()}
          </span>
          <h2 className="font-display text-primary mt-4 max-w-[20ch] text-[clamp(34px,4.2vw,56px)] leading-[1.05] font-light tracking-[-0.025em]">
            {m.home_intro_title_lead()}{" "}
            <em className="italic-display text-secondary">{m.home_intro_title_em()}</em>
          </h2>
        </div>
        <div className="text-on-surface-variant grid max-w-[46ch] gap-4 justify-self-start text-base leading-relaxed md:justify-self-end">
          <p>{m.home_intro_p1()}</p>
          <p>{m.home_intro_p2()}</p>
          <p>{m.home_intro_p3()}</p>
          <p>{m.home_intro_p4()}</p>
        </div>
      </div>
    </section>
  )
}
