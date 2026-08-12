import { Button } from "@/components/ui/button"

interface PagerProps {
  page: number
  maxPage: number
  onPageChange: (next: number) => void
}

export function Pager({ page, maxPage, onPageChange }: PagerProps) {
  if (maxPage <= 1) return null

  return (
    <div className="border-outline-variant flex items-center justify-center gap-4 border-t px-5 py-3.5">
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={page <= 1}
        onClick={() => onPageChange(page - 1)}
      >
        ‹ Prev
      </Button>
      <span className="text-on-surface-variant text-[12.5px]">
        Page {page} of {maxPage}
      </span>
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={page >= maxPage}
        onClick={() => onPageChange(page + 1)}
      >
        Next ›
      </Button>
    </div>
  )
}
