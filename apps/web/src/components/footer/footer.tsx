import type { ReactNode } from "react"

import { m } from "@/paraglide/messages"

interface FooterLink {
  href: string
  label: string
  external?: boolean
}

function FooterColumn({ title, links }: { title: string; links: FooterLink[] }) {
  return (
    <div>
      <h5 className="mb-4 font-mono text-[11px] font-medium tracking-[0.22em] text-white uppercase">
        {title}
      </h5>
      <ul className="grid gap-2.5">
        {/* Keyed on the label, not the href: two entries can point at the same
            place ("Book direct" and "Long stays" both go to #book), and sharing
            a key lets React drop one of them. */}
        {links.map((link) => (
          <li key={link.label}>
            <a
              href={link.href}
              target={link.external ? "_blank" : undefined}
              rel={link.external ? "noopener noreferrer" : undefined}
              className="hover:text-secondary-fixed text-sm transition-colors"
            >
              {link.label}
            </a>
          </li>
        ))}
      </ul>
    </div>
  )
}

function FooterBrand(): ReactNode {
  return (
    <div className="md:col-span-2 lg:col-span-1">
      <div className="font-display text-3xl text-white italic">
        {m.nav_brand_main()}
        <small className="mt-2 block font-mono text-[10px] tracking-[0.32em] text-slate-300 uppercase not-italic">
          {m.nav_brand_sub()}
        </small>
      </div>
      <p className="mt-6 max-w-[32ch] text-sm leading-relaxed text-slate-300">
        {m.footer_brand_desc()}
      </p>
    </div>
  )
}

export default function Footer() {
  const collection: FooterLink[] = [
    { href: "/villa/casadana", label: m.nav_link_dana() },
    { href: "/villa/casacasay", label: m.nav_link_casay() },
    { href: "#story", label: m.footer_link_story() },
  ]

  const stay: FooterLink[] = [
    { href: "/villa/casadana#book", label: m.footer_link_book_direct() },
    { href: "/villa/casadana#book", label: m.footer_link_long_stays() },
    { href: "/villa/casadana#location", label: m.footer_link_getting_there() },
  ]

  const follow: FooterLink[] = [
    {
      href: "https://www.instagram.com/casa_dana_los_alcazares/",
      label: "Instagram",
      external: true,
    },
    {
      href: "https://www.facebook.com/groups/607622748355117/",
      label: "Facebook",
      external: true,
    },
    {
      href: "mailto:casadana.contact@gmail.com",
      label: "casadana.contact@gmail.com",
    },
  ]

  return (
    <footer className="bg-primary mt-auto w-full pt-24 pb-8 text-slate-300">
      <div className="mx-auto max-w-[1440px] px-6 md:px-10">
        <div className="grid gap-10 border-b border-white/10 pb-16 md:grid-cols-2 lg:grid-cols-[1.6fr_1fr_1fr_1fr]">
          <FooterBrand />
          <FooterColumn title={m.footer_col_collection()} links={collection} />
          <FooterColumn title={m.footer_col_stay()} links={stay} />
          <FooterColumn title={m.footer_col_follow()} links={follow} />
        </div>

        <div className="flex flex-col gap-4 pt-8 font-mono text-[10.5px] tracking-[0.22em] uppercase md:flex-row md:justify-between">
          <span>{m.footer_copyright()}</span>
          <span>
            {m.footer_built_with_care()} ·{" "}
            <a
              href="https://github.com/TheHikuro/"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-secondary-fixed transition-colors"
            >
              {m.footer_built_by()}
            </a>
          </span>
        </div>
      </div>
    </footer>
  )
}
