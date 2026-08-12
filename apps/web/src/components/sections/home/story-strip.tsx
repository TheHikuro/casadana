import FRONT_IMG from "@/assets/casadana/front2.jpeg"
import POOL_IMG from "@/assets/casadessy/pool1.jpeg"
import { m } from "@/paraglide/messages"

export default function StoryStrip() {
  return (
    <section id="story" className="mx-auto max-w-[1440px] px-6 py-24 md:px-10 md:py-36">
      <div className="grid items-center gap-12 md:grid-cols-2 md:gap-24">
        <div className="relative grid grid-cols-2 gap-3 md:pr-16">
          <img
            src={FRONT_IMG}
            alt={m.home_story_image_alt_facade()}
            className="aspect-[3/4] w-full object-cover"
          />
          <img
            src={POOL_IMG}
            alt={m.home_story_image_alt_pool()}
            className="mt-16 aspect-[3/4] w-full object-cover"
          />
          <div className="bg-secondary-fixed text-on-secondary-container editorial-shadow absolute right-0 bottom-0 flex h-24 w-24 -rotate-6 flex-col items-center justify-center rounded-full text-center md:right-5 md:h-32 md:w-32">
            <span className="font-display text-2xl italic md:text-3xl">
              {m.home_story_stamp_year()}
            </span>
            <span className="mt-1 font-mono text-[8.5px] tracking-[0.22em] uppercase">
              {m.home_story_stamp_label()}
            </span>
          </div>
        </div>

        <div>
          <span className="text-secondary inline-flex items-center gap-3 font-mono text-[11px] tracking-[0.22em] uppercase before:block before:h-px before:w-6 before:bg-current">
            {m.home_story_index()}
          </span>
          <h2 className="font-display text-primary mt-4 mb-6 text-[clamp(32px,3.6vw,48px)] leading-[1.1] font-light tracking-[-0.025em]">
            {m.home_story_title_lead()}{" "}
            <em className="italic-display block">{m.home_story_title_em()}</em>
          </h2>
          <div className="text-on-surface-variant grid max-w-[50ch] gap-4 leading-relaxed">
            <p>{m.home_story_p1()}</p>
            <p>{m.home_story_p2()}</p>
            <p>{m.home_story_p3()}</p>
            <p>{m.home_story_p4()}</p>
            <p>{m.home_story_p5()}</p>
            <p>{m.home_story_p6()}</p>
            <p>{m.home_story_p7()}</p>
          </div>
          <p className="font-display text-primary border-outline-variant mt-8 max-w-[38ch] border-l pl-5 text-xl leading-snug font-light italic">
            {m.home_story_closing()}
          </p>
          <div className="font-display text-primary mt-8 text-xl italic">
            {m.home_story_signed_name()}
            <small className="text-on-surface-variant mt-1 block font-mono text-[10.5px] tracking-[0.22em] uppercase not-italic">
              {m.home_story_signed_role()}
            </small>
          </div>
        </div>
      </div>
    </section>
  )
}
