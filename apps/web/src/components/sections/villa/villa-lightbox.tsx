import { X } from "lucide-react"
import { useEffect, useState } from "react"

import {
  Dialog,
  DialogBackdrop,
  DialogPopup,
  DialogPortal,
  DialogTitle,
} from "@/components/ui/dialog"
import { GalleryCategory, getCategoryLabel } from "@/constants/gallery-categories.const"
import type { GalleryEntry } from "@/constants/villas.const"
import { cn } from "@/lib/utils"
import { m } from "@/paraglide/messages"

import VillaPhotoViewer from "./villa-photo-viewer"

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
  // Position of the photo opened full screen from this grid, `null` when none is.
  const [photoIndex, setPhotoIndex] = useState<number | null>(null)

  useEffect(() => {
    if (!open) setPhotoIndex(null)
  }, [open])

  const categories = (Object.keys(images) as Array<GalleryCategory>).filter(
    (c) => images[c].length > 0,
  )
  const tiles = images[category] ?? []

  const selectCategory = (next: GalleryCategory) => {
    setPhotoIndex(null)
    onCategory(next)
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) onClose()
      }}
    >
      <DialogPortal>
        <DialogBackdrop className="bg-inverse-surface/97 z-[100] opacity-100 transition-opacity duration-300 data-[ending-style]:opacity-0 data-[starting-style]:opacity-0" />
        <DialogPopup
          aria-modal="true"
          className="text-inverse-on-surface fixed inset-0 z-[100] flex flex-col opacity-100 transition-opacity duration-300 outline-none data-[ending-style]:opacity-0 data-[starting-style]:opacity-0"
        >
          <div className="border-outline-variant/20 flex flex-wrap items-center justify-between gap-4 border-b px-6 py-5 md:px-10">
            <DialogTitle className="font-display text-inverse-on-surface mb-0 flex-shrink-0 text-[18px] font-normal italic">
              {brand}
            </DialogTitle>
            <div className="flex flex-wrap gap-2">
              {categories.map((c) => (
                <button
                  key={c}
                  type="button"
                  onClick={() => selectCategory(c)}
                  className={cn(
                    "rounded-full border px-3.5 py-2 font-mono text-[10.5px] tracking-[0.18em] uppercase transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-current",
                    c === category
                      ? "text-primary border-inverse-on-surface bg-inverse-on-surface"
                      : "border-outline-variant/30 text-inverse-on-surface hover:border-outline-variant/60",
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
              className="border-outline-variant/40 text-inverse-on-surface hover:bg-inverse-on-surface/10 hover:border-outline-variant inline-flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-full border transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-current"
            >
              <X size={18} />
            </button>
          </div>

          <div className="flex-1 overflow-auto px-6 py-8 pb-16 md:px-10">
            <div className="mx-auto grid max-w-[1440px] auto-rows-[200px] grid-cols-1 gap-2.5 md:auto-rows-[240px] md:grid-cols-12">
              {tiles.map((img, i) => (
                <button
                  key={`${img.src}-${i}`}
                  type="button"
                  onClick={() => setPhotoIndex(i)}
                  className={cn(
                    "group relative cursor-zoom-in overflow-hidden text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-current",
                    spanClassFor(i, tiles.length),
                  )}
                >
                  <img
                    src={img.src}
                    alt={img.label}
                    className="h-full w-full object-cover transition-transform duration-700 group-hover:scale-[1.04] group-focus-visible:scale-[1.04]"
                  />
                  <span
                    className="absolute bottom-3.5 left-4 font-mono text-[10.5px] tracking-[0.18em] text-white uppercase"
                    style={{ textShadow: "0 2px 8px rgba(0,0,0,0.5)" }}
                  >
                    {img.label}
                  </span>
                </button>
              ))}
            </div>
          </div>

          <VillaPhotoViewer
            photos={tiles}
            index={photoIndex}
            onIndexChange={setPhotoIndex}
            onClose={() => setPhotoIndex(null)}
          />
        </DialogPopup>
      </DialogPortal>
    </Dialog>
  )
}
