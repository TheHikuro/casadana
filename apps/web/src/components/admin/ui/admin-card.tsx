import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

interface AdminCardProps {
  title: string
  /** Muted one-liner under the title explaining what the card controls. */
  sub?: string
  /** Right-aligned control in the card header, e.g. an "Ajouter une règle" button. */
  action?: ReactNode
  /** Set when the body supplies its own padding (tables, row lists). */
  flush?: boolean
  className?: string
  children: ReactNode
}

export function AdminCard({ title, sub, action, flush, className, children }: AdminCardProps) {
  return (
    <div className={cn("border-outline-variant bg-surface rounded-lg border", className)}>
      <div className="border-outline-variant flex flex-wrap items-center justify-between gap-3 border-b px-5 py-4">
        <div>
          <h3 className="text-on-surface text-[14.5px] font-semibold">{title}</h3>
          {sub && <p className="text-on-surface-variant mt-0.5 text-[12.5px]">{sub}</p>}
        </div>
        {action}
      </div>
      <div className={flush ? undefined : "p-5"}>{children}</div>
    </div>
  )
}
