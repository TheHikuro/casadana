import { ArrowRight, ArrowUpRight } from "lucide-react"
import { useMemo, useState } from "react"

import { GalleryCategory } from "@/constants/gallery-categories.const"
import type { BentoTile, VillaData } from "@/constants/villas.const"
import { cn } from "@/lib/utils"
import { m } from "@/paraglide/messages"

import VillaPhotoViewer from "./villa-photo-viewer"

interface VillaGalleryBentoProps {
  data: VillaData["gallery"]
  onOpenAll: () => void
}

const SPAN_CLASSES: Record<BentoTile["span"], string> = {
  wide: "md:col-span-7 md:row-span-2",
  tall: "md:col-span-5 md:row-span-2",
  half: "md:col-span-5",
  third: "md:col-span-4",
  quarter: "md:col-span-3",
  wideFlat: "md:col-span-8",
}

function PlaceholderArt({ label }: { label: string }) {
  return (
    <div
      className="absolute inset-0 flex items-center justify-center text-center"
      style={{
        background: "linear-gradient(135deg, oklch(85% 0.04 230) 0%, oklch(78% 0.06 220) 100%)",
      }}
    >
      <div
        className="pointer-events-none absolute inset-7 border border-dashed opacity-40"
        style={{ color: "oklch(35% 0.07 230)" }}
      />
      <div
        className="font-display flex flex-col items-center italic"
        style={{ color: "oklch(35% 0.07 230)" }}
      >
        <span className="text-[32px] leading-none">{label}</span>
        <span className="mt-2 font-mono text-[10px] tracking-[0.22em] uppercase not-italic opacity-70">
          {m.villa_gallery_photo_coming_soon()}
        </span>
      </div>
    </div>
  )
}

export default function VillaGalleryBento({ data, onOpenAll }: VillaGalleryBentoProps) {
  // The open photo is identified by its category + position, so the viewer keeps
  // working whatever the number of photos in each category is.
  const [viewer, setViewer] = useState<{ category: GalleryCategory; index: number } | null>(null)
  const viewerCategory = viewer?.category

  const viewerPhotos = useMemo(
    () => (viewerCategory ? (data.images[viewerCategory] ?? []) : []),
    [viewerCategory, data.images],
  )

  const photosFor = (category: GalleryCategory) => data.images[category] ?? []

  const openTile = (tile: BentoTile) => {
    const photos = photosFor(tile.category)
    if (photos.length === 0) return
    const startIndex = photos.findIndex((photo) => photo.src === tile.src)
    setViewer({ category: tile.category, index: startIndex === -1 ? 0 : startIndex })
  }

  return (
    <section id="gallery" className="bg-surface-container-low py-20 md:py-[140px]">
      <div className="mx-auto max-w-[1440px] px-6 md:px-10">
        <div className="mb-12 grid items-end gap-6 md:mb-16 md:grid-cols-[auto_1fr] md:gap-10">
          <div>
            <span className="text-secondary inline-flex items-center gap-3 font-mono text-[11px] tracking-[0.22em] uppercase before:block before:h-px before:w-6 before:bg-current">
              {data.chapter}
            </span>
            <h2 className="font-display text-primary mt-4 text-[clamp(40px,5.4vw,72px)] leading-none font-light tracking-[-0.025em]">
              <em className="italic-display">{data.titleItalic}</em>
              {data.titleTail}
            </h2>
          </div>
          <p className="text-on-surface-variant max-w-[44ch] justify-self-start text-[15px] leading-relaxed md:justify-self-end md:text-right">
            {data.description}
          </p>
        </div>

        <div className="grid auto-rows-[220px] grid-cols-1 gap-3 md:grid-cols-12">
          {data.tiles.map((tile, i) => {
            const canOpen = photosFor(tile.category).length > 0
            const visual = tile.placeholder ? (
              <PlaceholderArt label={tile.placeholderLabel ?? tile.caption} />
            ) : (
              <>
                <img
                  src={tile.src}
                  alt={tile.caption}
                  className="absolute inset-0 h-full w-full object-cover transition-transform duration-700 group-hover:scale-[1.06] group-focus-visible:scale-[1.06]"
                  style={{ willChange: "transform" }}
                />
                <div
                  className="absolute inset-0"
                  style={{
                    background:
                      "linear-gradient(180deg, transparent 45%, oklch(23.6% 0.108 253 / 0.7) 100%)",
                  }}
                />
              </>
            )

            const caption = (
              <div className="relative z-10 flex h-full flex-col justify-end p-5 text-white">
                <div className="flex items-end justify-between gap-4">
                  <div>
                    <div className="font-mono text-[10.5px] tracking-[0.22em] uppercase opacity-85">
                      {tile.index}
                    </div>
                    <div className="font-display mt-1.5 text-[26px] leading-tight italic md:text-[28px]">
                      {tile.caption}
                    </div>
                  </div>
                  {canOpen && (
                    <span className="group-hover:text-primary group-focus-visible:text-primary inline-flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-full border border-white/50 text-white transition-colors group-hover:bg-white group-focus-visible:bg-white">
                      <ArrowUpRight size={14} />
                    </span>
                  )}
                </div>
              </div>
            )

            const spanClass = SPAN_CLASSES[tile.span]

            // Tiles without any photo behind them stay non-interactive rather than
            // becoming a focusable control that does nothing.
            if (!canOpen) {
              return (
                <div
                  key={`${tile.label}-${i}`}
                  className={cn("group relative overflow-hidden text-left", spanClass)}
                >
                  {visual}
                  {caption}
                </div>
              )
            }

            return (
              <button
                key={`${tile.label}-${i}`}
                type="button"
                onClick={() => openTile(tile)}
                className={cn(
                  "group focus-visible:outline-primary relative cursor-zoom-in overflow-hidden text-left focus-visible:outline-2 focus-visible:outline-offset-2",
                  spanClass,
                )}
              >
                {visual}
                {caption}
              </button>
            )
          })}
        </div>

        <div className="mt-8 flex justify-end">
          <button
            type="button"
            onClick={onOpenAll}
            className="border-outline text-primary hover:bg-primary hover:text-on-primary hover:border-primary inline-flex items-center gap-3 rounded-full border px-6 py-3.5 font-mono text-[11px] tracking-[0.22em] uppercase transition-colors"
          >
            {data.totalLabel}
            <ArrowRight size={12} />
          </button>
        </div>
      </div>

      <VillaPhotoViewer
        photos={viewerPhotos}
        index={viewer?.index ?? null}
        onIndexChange={(index) =>
          setViewer((current) => (current ? { ...current, index } : current))
        }
        onClose={() => setViewer(null)}
      />
    </section>
  )
}
