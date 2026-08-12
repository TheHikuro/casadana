import { useGetVillaPricingSettings } from "@casa-dana/api"
import { Link } from "@tanstack/react-router"
import { ArrowRight } from "lucide-react"
import { DynamicIcon, type IconName } from "lucide-react/dynamic"

import { GalleryCategory } from "@/constants/gallery-categories.const"
import { cn } from "@/lib/utils"
import { m } from "@/paraglide/messages"

export interface GalleryImage {
  src: string
  alt: string
  label: string
  size?: "large" | "medium" | "small"
  category?: GalleryCategory
}

export interface PropertyCardProps {
  id: string
  badge: string
  category: string
  titlePrefix: string
  titleName: string
  subtitle: string
  /** One-line tagline shown above the body copy. */
  lead: string
  /** Body copy, one entry per rendered paragraph. */
  description: Array<string>
  exploreLabel: string
  price: { amount: number; currency: string }
  rating: { score: number; count: number }
  imageUrl: string
  imageAlt: string
  layout: "left" | "right"
  features: Array<{ icon: IconName; label: string }>
  galleryImages?: GalleryImage[]
}

function PropertyImage({
  imageUrl,
  imageAlt,
  badge,
  villaId,
}: Pick<PropertyCardProps, "imageUrl" | "imageAlt" | "badge"> & { villaId: string }) {
  return (
    <div className="relative overflow-hidden">
      <Link
        to="/villa/$villaId"
        params={{ villaId }}
        className="editorial-shadow group block aspect-[16/10] overflow-hidden"
      >
        <img
          src={imageUrl}
          alt={imageAlt}
          className="h-full w-full object-cover transition-transform duration-700 ease-out group-hover:scale-[1.04]"
        />
      </Link>
      <div className="bg-background text-primary editorial-shadow absolute top-6 left-6 px-4 py-2.5 font-mono text-[10.5px] tracking-[0.22em] uppercase">
        {badge}
      </div>
    </div>
  )
}

function PropertyFeatures({ features }: { features: PropertyCardProps["features"] }) {
  return (
    <div className="border-outline-variant grid grid-cols-1 gap-x-10 gap-y-4 border-y py-5 md:grid-cols-3">
      {features.map((feature) => (
        <div
          key={feature.label}
          className="flex items-center gap-2.5 font-mono text-[11px] tracking-[0.18em] uppercase"
        >
          <DynamicIcon name={feature.icon} size={16} strokeWidth={1.5} className="text-secondary" />
          {feature.label}
        </div>
      ))}
    </div>
  )
}

function PropertyMeta({
  villaId,
  price,
  rating,
}: Pick<PropertyCardProps, "price" | "rating"> & { villaId: string }) {
  // The nightly rate is edited per villa in the back-office (same source as the
  // villa page's booking panel). `price.amount` is only a fallback: settings
  // read back all-zero for a villa that has never been configured.
  const { data: settings } = useGetVillaPricingSettings(villaId)
  const amount = settings?.base_price_cents
    ? Math.round(settings.base_price_cents / 100)
    : price.amount

  return (
    <div className="flex flex-col items-start justify-between gap-3 md:flex-row md:items-baseline">
      <div className="font-display text-primary text-[26px] italic">
        {m.prop_price_from()} {price.currency}
        {amount}
        <small className="text-on-surface-variant ml-1 font-sans text-[13px] not-italic">
          {m.prop_price_per_night()}
        </small>
      </div>
      <div className="text-on-surface-variant font-mono text-[11px] tracking-[0.15em]">
        <span className="text-secondary mr-1 tracking-[2px]">★★★★★</span>
        {rating.score.toFixed(2)} · {m.prop_rating_reviews({ count: rating.count })}
      </div>
    </div>
  )
}

export default function PropertyCard({
  id,
  badge,
  category,
  titlePrefix,
  titleName,
  subtitle,
  lead,
  description,
  exploreLabel,
  price,
  rating,
  imageUrl,
  imageAlt,
  features,
  layout,
}: PropertyCardProps) {
  const isLeft = layout === "left"

  return (
    <section className={cn("py-20", !isLeft && "bg-surface-container-low")}>
      <div className="mx-auto max-w-[1440px] px-6 md:px-10">
        <div
          className={cn(
            "grid items-center gap-10 md:gap-16",
            isLeft ? "md:grid-cols-[7fr_5fr]" : "md:grid-cols-[5fr_7fr]",
          )}
        >
          <div className={cn(isLeft ? "order-1" : "order-1 md:order-2")}>
            <PropertyImage imageUrl={imageUrl} imageAlt={imageAlt} badge={badge} villaId={id} />
          </div>

          <div className={cn("order-2 space-y-6", isLeft ? "md:pl-8" : "md:order-1 md:pr-8")}>
            <span className="bg-secondary-container text-on-secondary-container inline-block rounded-full px-3.5 py-1.5 font-mono text-[11px] tracking-[0.22em] uppercase">
              {category}
            </span>

            <h3 className="font-display text-primary text-[clamp(36px,4.4vw,60px)] leading-[1.02] font-light tracking-[-0.025em]">
              {titlePrefix} <em className="italic-display">{titleName}</em>
              <br />
              {subtitle}
            </h3>

            <p className="font-display text-on-surface max-w-[34ch] text-xl leading-snug font-light tracking-[-0.01em]">
              {lead}
            </p>

            <div className="text-on-surface-variant grid max-w-[50ch] gap-3.5 text-[15.5px] leading-relaxed">
              {description.map((paragraph) => (
                <p key={paragraph}>{paragraph}</p>
              ))}
            </div>

            <PropertyFeatures features={features} />

            <PropertyMeta villaId={id} price={price} rating={rating} />

            <Link
              to="/villa/$villaId"
              params={{ villaId: id }}
              className="bg-primary text-on-primary hover:bg-primary-container inline-flex items-center gap-3 px-7 py-[18px] font-mono text-[11px] tracking-[0.28em] uppercase transition-colors"
            >
              {exploreLabel}
              <ArrowRight size={14} />
            </Link>
          </div>
        </div>
      </div>
    </section>
  )
}
