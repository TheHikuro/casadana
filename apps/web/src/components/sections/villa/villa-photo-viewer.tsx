import { ChevronLeft, ChevronRight, X } from "lucide-react"
import { useCallback, useEffect, useRef } from "react"

import {
  Dialog,
  DialogBackdrop,
  DialogPopup,
  DialogPortal,
  DialogTitle,
} from "@/components/ui/dialog"
import { m } from "@/paraglide/messages"

/** Minimal shape the viewer needs — `GalleryEntry` and `BentoTile` both satisfy it. */
export interface ViewerPhoto {
  src: string
  label: string
}

interface VillaPhotoViewerProps {
  /** The current set the arrows navigate through. May be empty (the viewer then stays closed). */
  photos: Array<ViewerPhoto>
  /** Index of the displayed photo within `photos`, or `null` when closed. */
  index: number | null
  onIndexChange: (index: number) => void
  onClose: () => void
}

const SWIPE_THRESHOLD_PX = 48

/** Wraps around so navigation keeps working whatever the set size is. */
function wrap(index: number, total: number): number {
  return ((index % total) + total) % total
}

const CONTROL_CLASS =
  "border-outline-variant/40 text-inverse-on-surface hover:border-outline-variant hover:bg-inverse-on-surface/10 inline-flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-full border backdrop-blur-sm transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-current"

/**
 * Full-screen single-photo overlay: `object-contain` so nothing is cropped,
 * arrow-key / arrow-button / swipe navigation, and Escape or a background click
 * to close. Focus is trapped and restored by the underlying Dialog primitive.
 */
export default function VillaPhotoViewer({
  photos,
  index,
  onIndexChange,
  onClose,
}: VillaPhotoViewerProps) {
  const total = photos.length
  const open = index !== null && total > 0
  const currentIndex = open ? wrap(index, total) : 0
  const current = open ? photos[currentIndex] : undefined
  const hasSiblings = total > 1

  const go = useCallback(
    (delta: number) => {
      if (index === null || total === 0) return
      onIndexChange(wrap(index + delta, total))
    },
    [index, total, onIndexChange],
  )

  // Left/right arrows move through the set. Escape is handled by the Dialog.
  useEffect(() => {
    if (!open || !hasSiblings) return
    const handler = (event: KeyboardEvent) => {
      if (event.key === "ArrowLeft") {
        event.preventDefault()
        go(-1)
      } else if (event.key === "ArrowRight") {
        event.preventDefault()
        go(1)
      }
    }
    document.addEventListener("keydown", handler)
    return () => document.removeEventListener("keydown", handler)
  }, [open, hasSiblings, go])

  // Warm the neighbours so stepping through a large gallery does not flash.
  useEffect(() => {
    if (!open || !hasSiblings) return
    for (const offset of [-1, 1]) {
      const neighbour = photos[wrap(currentIndex + offset, total)]
      if (!neighbour) continue
      const preloader = new Image()
      preloader.src = neighbour.src
    }
  }, [open, hasSiblings, photos, currentIndex, total])

  const touchStartX = useRef<number | null>(null)

  const handleTouchStart = (event: React.TouchEvent) => {
    touchStartX.current = event.touches[0]?.clientX ?? null
  }

  const handleTouchEnd = (event: React.TouchEvent) => {
    const startX = touchStartX.current
    touchStartX.current = null
    const endX = event.changedTouches[0]?.clientX
    if (startX === null || endX === undefined) return
    const deltaX = endX - startX
    if (Math.abs(deltaX) < SWIPE_THRESHOLD_PX) return
    go(deltaX < 0 ? 1 : -1)
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) onClose()
      }}
    >
      <DialogPortal>
        <DialogBackdrop
          forceRender
          className="bg-inverse-surface/95 z-[200] opacity-100 backdrop-blur-md transition-opacity duration-300 data-[ending-style]:opacity-0 data-[starting-style]:opacity-0"
        />
        <DialogPopup
          aria-modal="true"
          className="text-inverse-on-surface fixed inset-0 z-[200] flex flex-col opacity-100 transition-opacity duration-300 outline-none data-[ending-style]:opacity-0 data-[starting-style]:opacity-0"
          onTouchStart={handleTouchStart}
          onTouchEnd={handleTouchEnd}
        >
          <header className="border-outline-variant/25 relative z-10 flex items-center justify-between gap-4 border-b px-5 py-4 md:px-10">
            <DialogTitle className="sr-only">{current?.label}</DialogTitle>
            <span className="text-inverse-on-surface/55 font-mono text-[10.5px] tracking-[0.22em] uppercase tabular-nums">
              {hasSiblings
                ? `${String(currentIndex + 1).padStart(2, "0")} / ${String(total).padStart(2, "0")}`
                : m.gallery_title()}
            </span>
            <button
              type="button"
              onClick={onClose}
              aria-label={m.gallery_close()}
              className={CONTROL_CLASS}
            >
              <X size={18} />
            </button>
          </header>

          {/* Clicking the empty area around the photo closes the viewer. */}
          <div
            className="relative flex flex-1 items-center justify-center overflow-hidden p-4 md:px-20 md:py-10"
            onClick={(event) => {
              if (event.target === event.currentTarget) onClose()
            }}
          >
            {current && (
              <img
                key={current.src}
                src={current.src}
                alt={current.label}
                className="border-outline-variant/20 max-h-full max-w-full border object-contain"
              />
            )}

            {hasSiblings && (
              <>
                <button
                  type="button"
                  onClick={() => go(-1)}
                  aria-label={m.villa_photo_viewer_previous()}
                  className={`${CONTROL_CLASS} bg-inverse-surface/50 absolute top-1/2 left-3 -translate-y-1/2 md:left-6`}
                >
                  <ChevronLeft size={20} />
                </button>
                <button
                  type="button"
                  onClick={() => go(1)}
                  aria-label={m.villa_photo_viewer_next()}
                  className={`${CONTROL_CLASS} bg-inverse-surface/50 absolute top-1/2 right-3 -translate-y-1/2 md:right-6`}
                >
                  <ChevronRight size={20} />
                </button>
              </>
            )}
          </div>

          <footer className="border-outline-variant/25 relative z-10 flex items-center justify-center border-t px-5 py-4 md:px-10">
            <span className="font-display text-inverse-on-surface/90 truncate text-[18px] italic md:text-[20px]">
              {current?.label}
            </span>
          </footer>
        </DialogPopup>
      </DialogPortal>
    </Dialog>
  )
}
