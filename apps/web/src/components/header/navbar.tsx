import { Link } from "@tanstack/react-router"
import { ArrowRight } from "lucide-react"
import { useEffect, useState } from "react"

import { cn } from "@/lib/utils"
import { m } from "@/paraglide/messages"

import { LangSelector } from "../lang-selector"

interface NavLink {
  href: string
  label: string
}

function NavBrand() {
  return (
    <Link to="/" className="font-display text-[22px] leading-none italic">
      {m.nav_brand_main()}
      <small className="mt-1 block font-mono text-[9px] tracking-[0.32em] uppercase not-italic opacity-70">
        {m.nav_brand_sub()}
      </small>
    </Link>
  )
}

function NavLinks({ links }: { links: NavLink[] }) {
  return (
    <nav className="hidden items-center justify-center gap-7 lg:flex">
      {links.map((link) => (
        <a
          key={link.href}
          href={link.href}
          className="group relative pb-1 font-mono text-[11px] tracking-[0.22em] uppercase opacity-90 transition-opacity hover:opacity-100"
        >
          {link.label}
          <span className="pointer-events-none absolute right-0 -bottom-0.5 left-0 h-px origin-left scale-x-0 bg-current transition-transform duration-300 group-hover:scale-x-100" />
        </a>
      ))}
    </nav>
  )
}

export default function Navbar() {
  const [isScrolled, setIsScrolled] = useState(false)

  useEffect(() => {
    const handleScroll = () => setIsScrolled(window.scrollY > 50)
    handleScroll()
    window.addEventListener("scroll", handleScroll, { passive: true })
    return () => window.removeEventListener("scroll", handleScroll)
  }, [])

  const links: NavLink[] = [
    { href: "/#collection", label: m.nav_link_collection() },
    { href: "/villa/casadana", label: m.nav_link_dana() },
    { href: "/villa/casacasay", label: m.nav_link_casay() },
    { href: "/#story", label: m.nav_link_story() },
  ]

  return (
    <header
      className={cn(
        "fixed inset-x-0 top-0 z-50 backdrop-blur-md transition-all duration-300",
        isScrolled
          ? "bg-background/95 text-primary shadow-[0_20px_40px_-20px_oklch(23.6%_0.108_253/0.12)]"
          : "bg-background/[0.18] text-white",
      )}
    >
      <div
        className={cn(
          "mx-auto grid max-w-[1440px] items-center gap-6 px-6 transition-all md:px-10",
          isScrolled ? "py-4" : "py-5",
          "grid-cols-[1fr_auto] lg:grid-cols-[1fr_auto_1fr]",
        )}
      >
        <NavBrand />
        <NavLinks links={links} />
        <div className="flex items-center justify-end gap-4">
          <div className="hidden md:block">
            <LangSelector />
          </div>
          <Link
            to="/villa/$villaId"
            params={{ villaId: "casadana" }}
            className={cn(
              "inline-flex items-center gap-2 rounded-full border px-4 py-2.5 text-[10.5px] tracking-[0.22em] uppercase transition-all md:px-4.5",
              isScrolled ? "bg-primary text-on-primary border-primary" : "border-current",
            )}
          >
            {m.nav_reserve()}
            <ArrowRight size={10} />
          </Link>
        </div>
      </div>
    </header>
  )
}
