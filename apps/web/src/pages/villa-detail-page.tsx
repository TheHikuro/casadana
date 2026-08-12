import { useParams } from "@tanstack/react-router"
import { useEffect, useState } from "react"

import VillaHero from "@/components/sections/villa-hero"
import VillaAbout from "@/components/sections/villa/villa-about"
import VillaCtaStrip from "@/components/sections/villa/villa-cta-strip"
import VillaExperiences from "@/components/sections/villa/villa-experiences"
import VillaFaq from "@/components/sections/villa/villa-faq"
import VillaGalleryBento from "@/components/sections/villa/villa-gallery-bento"
import VillaLightbox from "@/components/sections/villa/villa-lightbox"
import VillaLocalArea from "@/components/sections/villa/villa-local-area"
import VillaReviews from "@/components/sections/villa/villa-reviews"
import VillaRibbon from "@/components/sections/villa/villa-ribbon"
import VillaSister from "@/components/sections/villa/villa-sister"
import { GalleryCategory } from "@/constants/gallery-categories.const"
import { getVilla } from "@/constants/villas.const"
import { m } from "@/paraglide/messages"

export default function VillaDetailPage() {
  const { villaId } = useParams({ from: "/_public/villa/$villaId" })
  const villa = getVilla(villaId)

  const [lbOpen, setLbOpen] = useState(false)
  const [lbCategory, setLbCategory] = useState<GalleryCategory>("LIVING_SPACES")

  useEffect(() => {
    window.scrollTo(0, 0)
    setLbOpen(false)
  }, [villaId])

  if (!villa) {
    return (
      <section className="mx-auto max-w-[1440px] px-6 py-32 md:px-10">
        <span className="text-secondary inline-flex items-center gap-3 font-mono text-[11px] tracking-[0.22em] uppercase before:block before:h-px before:w-6 before:bg-current">
          {m.villa_not_found_index()}
        </span>
        <h1 className="font-display text-primary mt-4 text-[clamp(40px,5vw,64px)] font-light italic">
          {m.villa_not_found_title()}
        </h1>
      </section>
    )
  }

  const openAll = () => {
    const firstCat = (Object.keys(villa.gallery.images) as GalleryCategory[]).find(
      (c) => villa.gallery.images[c].length > 0,
    )
    if (firstCat) {
      setLbCategory(firstCat)
      setLbOpen(true)
    }
  }

  return (
    <>
      <VillaHero villaSlug={villaId} {...villa.hero} />
      <VillaRibbon phrases={villa.ribbon} />
      <VillaAbout villaSlug={villaId} about={villa.about} booking={villa.booking} />
      <VillaGalleryBento data={villa.gallery} onOpenAll={openAll} />
      <VillaLocalArea data={villa.localArea} />
      <VillaExperiences data={villa.experiences} />
      <VillaReviews villaSlug={villaId} data={villa.reviews} />
      <VillaFaq data={villa.faq} />
      <VillaCtaStrip data={villa.ctaStrip} />
      <VillaSister data={villa.sister} />

      <VillaLightbox
        brand={`Casa ${villa.hero.titleItalic}`}
        open={lbOpen}
        category={lbCategory}
        images={villa.gallery.images}
        onCategory={setLbCategory}
        onClose={() => setLbOpen(false)}
      />
    </>
  )
}
