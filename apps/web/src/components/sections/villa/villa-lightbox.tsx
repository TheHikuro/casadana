import { X } from "lucide-react"
import { useEffect } from "react"

import {
  GalleryCategory,
  getCategoryLabel,
} from "@/constants/gallery-categories.const"
import type { GalleryEntry } from "@/constants/villas.const"
import { cn } from "@/lib/utils"
import { m } from "@/paraglide/messages"

interface VillaLightboxProps {
  brand: string
  open: boolean
  category: GalleryCategory
  images: Record<GalleryCategory, Array<GalleryEntry>>
  onCategory: (c: GalleryCategory) => void
  onClose: () => void
}

// Tiles are sized purely from their position, alternating a wide/narrow pair
// (8 + 4 = 12 columns, so every row fills edge-to-edge). A trailing odd tile
// takes the full row instead of leaving a half-filled gap.
function spanClassFor(index: number, total: number): string {
  if (index === total - 1 && total % 2 === 1) return "md:col-span-12"
  return index % 2 === 0 ? "md:col-span-8" : "md:col-span-4"
}

export default function VillaLightbox({
  brand,
  open,
  category,
  images,
  onCategory,
  onClose,
}: VillaLightboxProps) {
  useEffect(() => {
    if (!open) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose()
    }
    document.addEventListener("keydown", handler)
    document.body.style.overflow = "hidden"
    return () => {
      document.removeEventListener("keydown", handler)
      document.body.style.overflow = ""
    }
  }, [open, onClose])

  const categories = (Object.keys(images) as Array<GalleryCategory>).filter(
    (c) => images[c].length > 0,
  )
  const tiles = images[category] ?? []

  return (
    <div
      className={cn(
        "fixed inset-0 z-[100] flex flex-col transition-opacity duration-300",
        open ? "pointer-events-auto opacity-100" : "pointer-events-none opacity-0",
      )}
      style={{ background: "oklch(12.1% 0.012 60 / 0.97)" }}
      aria-hidden={!open}
    >
      <div className="flex flex-wrap items-center justify-between gap-4 border-b border-white/10 px-6 py-5 text-white md:px-10">
        <span className="font-display flex-shrink-0 text-[18px] italic">{brand}</span>
        <div className="flex flex-wrap gap-2">
          {categories.map((c) => (
            <button
              key={c}
              type="button"
              onClick={() => onCategory(c)}
              className={cn(
                "rounded-full border px-3.5 py-2 font-mono text-[10.5px] tracking-[0.18em] uppercase transition-colors",
                c === category
                  ? "text-primary border-white bg-white"
                  : "border-white/25 text-white hover:border-white/50",
              )}
            >
              {getCategoryLabel(c)}
            </button>
          ))}
        </div>
        <button
          type="button"
          onClick={onClose}
          aria-label={m.villa_lightbox_close()}
          className="inline-flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-full border border-white/40 text-white transition-colors hover:bg-white/10"
        >
          <X size={18} />
        </button>
      </div>

      <div className="flex-1 overflow-auto px-6 py-8 pb-16 md:px-10">
        <div className="mx-auto grid max-w-[1440px] auto-rows-[200px] grid-cols-1 gap-2.5 md:auto-rows-[240px] md:grid-cols-12">
          {tiles.map((img, i) => (
            <div
              key={i}
              className={cn(
                "relative overflow-hidden",
                spanClassFor(i, tiles.length),
              )}
            >
              <img
                src={img.src}
                alt={img.label}
                className="h-full w-full object-cover"
              />
              <div
                className="absolute bottom-3.5 left-4 font-mono text-[10.5px] tracking-[0.18em] uppercase text-white"
                style={{ textShadow: "0 2px 8px rgba(0,0,0,0.5)" }}
              >
                {img.label}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
